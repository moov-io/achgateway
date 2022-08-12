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

package pipeline

import (
	"context"
	"errors"
	"strings"

	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
)

// FileReceiver accepts an ACH file from a number of pubsub Subscriptions and
// finds the appropriate aggregator for the shardKey.
type FileReceiver struct {
	logger           log.Logger
	defaultShardName string

	shardRepository  shards.Repository
	shardAggregators map[string]*aggregator

	httpFiles   *pubsub.Subscription
	streamFiles *pubsub.Subscription

	transformConfig *models.TransformConfig
}

func newFileReceiver(
	logger log.Logger,
	defaultShardName string,
	shardRepository shards.Repository,
	shardAggregators map[string]*aggregator,
	httpFiles *pubsub.Subscription,
	streamFiles *pubsub.Subscription,
	transformConfig *models.TransformConfig,
) *FileReceiver {
	return &FileReceiver{
		logger:           logger,
		defaultShardName: defaultShardName,
		shardRepository:  shardRepository,
		shardAggregators: shardAggregators,
		httpFiles:        httpFiles,
		streamFiles:      streamFiles,
		transformConfig:  transformConfig,
	}
}

func (fr *FileReceiver) Start(ctx context.Context) {
	for {
		// Create a context that will be shutdown by its parent or after a read iteration
		innerCtx, cancelFunc := context.WithCancel(ctx)

		select {
		case err := <-fr.handleMessage(innerCtx, fr.httpFiles):
			incomingHTTPFiles.With().Add(1)
			if err != nil {
				httpFileProcessingErrors.With().Add(1)
				fr.logger.LogErrorf("error handling http file: %v", err)
			}

		case err := <-fr.handleMessage(innerCtx, fr.streamFiles):
			incomingStreamFiles.With().Add(1)
			if err != nil {
				streamFileProcessingErrors.With().Add(1)
				fr.logger.LogErrorf("error handling stream file: %v", err)
			}

		case <-ctx.Done():
			cancelFunc()
			fr.Shutdown()
			return
		}

		// After processing a message cancel the inner context to release any resources.
		cancelFunc()
	}
}

func (fr *FileReceiver) Shutdown() {
	fr.logger.Log("shutting down xfer aggregation")

	// Pass a context that we cancel right away to our subscriptions
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	if fr.httpFiles != nil {
		if err := fr.httpFiles.Shutdown(ctx); err != nil {
			fr.logger.LogErrorf("problem shutting down http file subscription: %v", err)
		}
	}
	if fr.streamFiles != nil {
		if err := fr.streamFiles.Shutdown(ctx); err != nil {
			fr.logger.LogErrorf("problem shutting down stream file subscription: %v", err)
		}
	}
}

func (fr *FileReceiver) RegisterAdminRoutes(r *admin.Server) {
	r.AddHandler("/trigger-cutoff", fr.triggerManualCutoff())

	r.AddHandler("/shards", fr.listShards())

	sub := r.Subrouter("/shards/{shardName}")
	sub.HandleFunc("/files", fr.listShardFiles())
	sub.PathPrefix("/files/{filepath}").Handler(fr.getShardFile())
}

// handleMessage will listen for an incoming.ACHFile to pass off to an aggregator for the shard
// responsible. It does so with a database lookup and the fixed set of Shards from the file config.
func (fr *FileReceiver) handleMessage(ctx context.Context, sub *pubsub.Subscription) chan error {
	out := make(chan error, 1)
	if sub == nil {
		return out
	}
	cleanup := func() {
		out <- nil
	}
	go func() {
		receiver := make(chan *pubsub.Message)
		go func() {
			msg, err := sub.Receive(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				if strings.Contains(err.Error(), "Subscription has been Shutdown") {
					return
				}
				if strings.Contains(err.Error(), "connect: ") {
					out <- err
				}
				fr.logger.LogErrorf("ERROR receiving message: %v", err)
			}
			receiver <- msg
		}()

		select {
		case msg := <-receiver:
			if msg != nil {
				out <- fr.processMessage(msg)
				return
			} else {
				cleanup()
				return
			}

		case <-ctx.Done():
			cleanup()
			return
		}
	}()
	return out
}

func (fr *FileReceiver) processMessage(msg *pubsub.Message) error {
	data := msg.Body
	var err error

	// Optionally decode and decrypt message
	data, err = compliance.Reveal(fr.transformConfig, data)
	if err != nil {
		fr.logger.Error().LogErrorf("unable to reveal event: %v", err)
		return nil
	}

	event, err := models.Read(data)
	if err != nil {
		fr.logger.Error().LogErrorf("unable to read event: %v", err)
		return nil
	}

	switch evt := event.Event.(type) {
	case incoming.ACHFile:
		err = fr.processACHFile(evt)
		if err != nil {
			return err
		}
		msg.Ack()
		return nil

	case *models.QueueACHFile:
		file := incoming.ACHFile(*evt)
		err = fr.processACHFile(file)
		if err != nil {
			return err
		}
		msg.Ack()
		return nil

	case *models.CancelACHFile:
		err = fr.cancelACHFile(evt)
		if err != nil {
			return err
		}
		msg.Ack()
		return nil
	}

	fr.logger.Error().LogErrorf("unexpected %T event", event.Event)
	return nil
}

func (fr *FileReceiver) getAggregator(shardKey string) *aggregator {
	shardName, err := fr.shardRepository.Lookup(shardKey)
	if err != nil {
		fr.logger.Error().LogErrorf("problem looking up shardKey=%s: %v", shardKey, err)
		return nil
	}

	agg, exists := fr.shardAggregators[shardName]
	if !exists {
		agg, exists = fr.shardAggregators[fr.defaultShardName]
		if !exists {
			filesMissingShardAggregators.With("shard", shardName).Add(1)
			fr.logger.Error().LogErrorf("missing shardAggregator for shardKey=%s shardName=%s", shardKey, shardName)
			return nil
		}
	}
	if agg == nil {
		fr.logger.Error().LogErrorf("nil shardAggregator for shardKey=%s shardName=%s", shardKey, shardName)
		return nil
	}
	return agg
}

func (fr *FileReceiver) processACHFile(file incoming.ACHFile) error {
	if file.FileID == "" || file.ShardKey == "" {
		return errors.New("missing fileID or shardKey")
	}

	err := file.Validate()
	if err != nil {
		fr.logger.Error().LogErrorf("invalid ACHFile: %v", err)
		return nil
	}

	agg := fr.getAggregator(file.ShardKey)
	if agg == nil {
		return nil
	}

	logger := fr.logger.With(log.Fields{
		"fileID":    log.String(file.FileID),
		"shardName": log.String(agg.shard.Name),
		"shardKey":  log.String(file.ShardKey),
	})
	logger.Log("begin handling of received ACH file")

	err = agg.acceptFile(file)
	if err != nil {
		return logger.Error().LogErrorf("problem accepting file under shardName=%s", agg.shard.Name).Err()
	}

	// Record the file as accepted
	pendingFiles.With("shard", agg.shard.Name).Add(1)
	logger.Log("finished handling ACH file")

	return nil
}

func (fr *FileReceiver) cancelACHFile(cancel *models.CancelACHFile) error {
	if cancel == nil || cancel.FileID == "" || cancel.ShardKey == "" {
		return errors.New("missing fileID or shardKey")
	}

	agg := fr.getAggregator(cancel.ShardKey)
	if agg == nil {
		return nil
	}

	logger := fr.logger.With(log.Fields{
		"fileID":    log.String(cancel.FileID),
		"shardName": log.String(agg.shard.Name),
		"shardKey":  log.String(cancel.ShardKey),
	})
	logger.Log("begin canceling ACH file")

	evt := incoming.CancelACHFile(*cancel)

	err := agg.cancelFile(evt)
	if err != nil {
		return logger.Error().LogErrorf("problem canceling file: %v", err).Err()
	}

	logger.Log("finished cancel of file")
	return nil
}
