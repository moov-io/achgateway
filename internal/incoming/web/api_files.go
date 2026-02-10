// Licensed to The Moov Authors under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. The Moov Authors licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/telemetry"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gocloud.dev/pubsub"
)

func NewFilesController(
	logger log.Logger,
	cfg service.HTTPConfig,
	pub stream.Publisher,
	queueFileResponses chan incoming.QueueACHFileResponse,
	cancellationResponses chan models.FileCancellationResponse,
) *FilesController {
	controller := &FilesController{
		logger:    logger,
		cfg:       cfg,
		publisher: pub,

		activeQueueFiles:   make(map[string]chan incoming.QueueACHFileResponse),
		queueFileResponses: queueFileResponses,

		activeCancellations:   make(map[string]chan models.FileCancellationResponse),
		cancellationResponses: cancellationResponses,
	}

	controller.listenForQueueACHFileResponses()
	controller.listenForCancellations()

	return controller
}

type FilesController struct {
	logger    log.Logger
	cfg       service.HTTPConfig
	publisher stream.Publisher

	queueFileLock      sync.Mutex
	activeQueueFiles   map[string]chan incoming.QueueACHFileResponse
	queueFileResponses chan incoming.QueueACHFileResponse

	cancellationLock      sync.Mutex
	activeCancellations   map[string]chan models.FileCancellationResponse
	cancellationResponses chan models.FileCancellationResponse
}

func (c *FilesController) listenForQueueACHFileResponses() {
	c.logger.Info().Log("listening for QueueACHFile responses")
	go func() {
		for {
			// Wait for a message
			resp := <-c.queueFileResponses
			logger := c.logger.Info().With(log.Fields{
				"file_id":   log.String(resp.FileID),
				"shard_key": log.String(resp.ShardKey),
			})

			if resp.Error != "" {
				logger.Error().LogErrorf("problem with QueueACHFile: %v", resp.Error)
			} else {
				logger.Info().Logf("received QueueACHFile response")
			}

			fileID := strings.TrimSuffix(resp.FileID, ".ach")

			c.queueFileLock.Lock()
			out, exists := c.activeQueueFiles[fileID]
			if exists {
				out <- resp
				delete(c.activeQueueFiles, fileID)
			}
			c.queueFileLock.Unlock()
		}
	}()
}

func (c *FilesController) listenForCancellations() {
	c.logger.Info().Log("listening for cancellation responses")
	go func() {
		for {
			// Wait for a message
			cancel := <-c.cancellationResponses
			c.logger.Info().With(log.Fields{
				"file_id":    log.String(cancel.FileID),
				"shard_key":  log.String(cancel.ShardKey),
				"successful": log.Bool(cancel.Successful),
			}).Log("received cancellation response")

			fileID := strings.TrimSuffix(cancel.FileID, ".ach")

			c.cancellationLock.Lock()
			out, exists := c.activeCancellations[fileID]
			if exists {
				out <- cancel
				delete(c.activeCancellations, fileID)
			}
			c.cancellationLock.Unlock()
		}
	}()
}

func (c *FilesController) AppendRoutes(router *mux.Router) *mux.Router {
	router.
		Name("Files.create").
		Methods("POST").
		Path("/shards/{shardKey}/files/{fileID}").
		HandlerFunc(c.CreateFileHandler)

	router.
		Name("Files.cancel").
		Methods("DELETE").
		Path("/shards/{shardKey}/files/{fileID}").
		HandlerFunc(c.CancelFileHandler)

	return router
}

func (c *FilesController) CreateFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shardKey, fileID := vars["shardKey"], vars["fileID"]
	if shardKey == "" || fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, span := telemetry.StartSpan(r.Context(), "create-file-handler", trace.WithAttributes(
		attribute.String("achgateway.shardKey", shardKey),
		attribute.String("achgateway.fileID", fileID),
	))
	defer span.End()

	logger := c.logger.With(log.Fields{
		"shard_key": log.String(shardKey),
		"file_id":   log.String(fileID),
	})

	bs, err := c.readBody(r)
	if err != nil {
		logger.Error().LogErrorf("error reading file: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, err := ach.NewReader(bytes.NewReader(bs)).Read()
	if err != nil {
		// attempt JSON decode
		f, err := ach.FileFromJSON(bs)
		if f == nil || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file = *f
	}

	waiter := make(chan incoming.QueueACHFileResponse, 1)
	if err := c.publishFile(ctx, shardKey, fileID, &file, waiter); err != nil {
		logger.Error().LogErrorf("publishing file: %v", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var response incoming.QueueACHFileResponse
	select {
	case resp := <-waiter:
		response = resp

	case <-time.After(10 * time.Second):
		response = incoming.QueueACHFileResponse{
			FileID:   fileID,
			ShardKey: shardKey,
			Error:    "timeout exceeded",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (c *FilesController) readBody(req *http.Request) ([]byte, error) {
	defer req.Body.Close()

	var reader io.Reader = req.Body
	if c.cfg.MaxBodyBytes > 0 {
		reader = io.LimitReader(reader, c.cfg.MaxBodyBytes)
	}
	bs, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return compliance.Reveal(c.cfg.Transform, bs)
}

func (c *FilesController) publishFile(ctx context.Context, shardKey, fileID string, file *ach.File, waiter chan incoming.QueueACHFileResponse) error {
	c.queueFileLock.Lock()
	c.activeQueueFiles[fileID] = waiter
	c.queueFileLock.Unlock()

	bs, err := compliance.Protect(c.cfg.Transform, models.Event{
		Event: incoming.ACHFile{
			FileID:   fileID,
			ShardKey: shardKey,
			File:     file,
		},
	})
	if err != nil {
		return fmt.Errorf("unable to protect incoming file event: %v", err)
	}

	meta := make(map[string]string)
	meta["fileID"] = fileID
	meta["shardKey"] = shardKey

	return c.publisher.Send(ctx, &pubsub.Message{
		Body:     bs,
		Metadata: meta,
	})
}

func (c *FilesController) CancelFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shardKey, fileID := vars["shardKey"], vars["fileID"]
	if shardKey == "" || fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Remove .ach suffix if the request added it
	fileID = strings.TrimSuffix(fileID, ".ach")

	ctx, span := telemetry.StartSpan(r.Context(), "cancel-file-handler", trace.WithAttributes(
		attribute.String("achgateway.shardKey", shardKey),
		attribute.String("achgateway.fileID", fileID),
	))
	defer span.End()

	waiter := make(chan models.FileCancellationResponse, 1)
	err := c.cancelFile(ctx, shardKey, fileID, waiter)
	if err != nil {
		c.logger.With(log.Fields{
			"shard_key": log.String(shardKey),
			"file_id":   log.String(fileID),
		}).Error().LogErrorf("canceling file: %v", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var response models.FileCancellationResponse
	select {
	case resp := <-waiter:
		response = resp

	case <-time.After(10 * time.Second):
		response = models.FileCancellationResponse{
			FileID:     fileID,
			ShardKey:   shardKey,
			Successful: false,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (c *FilesController) cancelFile(ctx context.Context, shardKey, fileID string, waiter chan models.FileCancellationResponse) error {
	c.cancellationLock.Lock()
	c.activeCancellations[fileID] = waiter
	c.cancellationLock.Unlock()

	bs, err := compliance.Protect(c.cfg.Transform, models.Event{
		Event: incoming.CancelACHFile{
			FileID:   fileID,
			ShardKey: shardKey,
		},
	})
	if err != nil {
		return fmt.Errorf("unable to protect cancel file event: %v", err)
	}

	meta := make(map[string]string)
	meta["fileID"] = fileID
	meta["shardKey"] = shardKey

	return c.publisher.Send(ctx, &pubsub.Message{
		Body:     bs,
		Metadata: meta,
	})
}
