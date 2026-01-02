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
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/files"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/telemetry"

	"github.com/IBM/sarama"
	"gocloud.dev/pubsub"
)

// FileReceiver accepts an ACH file from a number of pubsub Subscriptions and
// finds the appropriate aggregator for the shardKey.
type FileReceiver struct {
	logger log.Logger
	cfg    *service.Config

	mu     sync.RWMutex
	cancel context.CancelFunc

	defaultShardName string

	eventEmitter events.Emitter

	shardRepository  shards.Repository
	shardAggregators map[string]*aggregator

	fileRepository files.Repository

	httpFiles   stream.Subscription
	streamFiles stream.Subscription

	QueueFileResponses    chan incoming.QueueACHFileResponse
	CancellationResponses chan models.FileCancellationResponse

	transformConfig *models.TransformConfig
}

func newFileReceiver(
	logger log.Logger,
	cfg *service.Config,
	eventEmitter events.Emitter,
	shardRepository shards.Repository,
	shardAggregators map[string]*aggregator,
	fileRepository files.Repository,
	httpFiles stream.Subscription,
	transformConfig *models.TransformConfig,
) (*FileReceiver, error) {
	// Create FileReceiver and connect streamFiles
	fr := &FileReceiver{
		logger:                logger,
		cfg:                   cfg,
		eventEmitter:          eventEmitter,
		defaultShardName:      cfg.Sharding.Default,
		shardRepository:       shardRepository,
		shardAggregators:      shardAggregators,
		fileRepository:        fileRepository,
		httpFiles:             httpFiles,
		QueueFileResponses:    make(chan incoming.QueueACHFileResponse, 1000),
		CancellationResponses: make(chan models.FileCancellationResponse, 1000),
		transformConfig:       transformConfig,
	}
	err := fr.reconnect()
	if err != nil {
		return nil, err
	}
	return fr, nil
}

func (fr *FileReceiver) reconnect() error {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	// Close any existing subscription
	if fr.streamFiles != nil {
		fr.streamFiles.Shutdown(context.Background())
	}

	streamSub, err := stream.OpenSubscription(fr.logger, fr.cfg)
	if err != nil {
		return fmt.Errorf("creating stream subscription: %v", err)
	}
	fr.streamFiles = streamSub

	return nil
}

func (fr *FileReceiver) Start(ctx context.Context) {
	for {
		// Create a context that will be shutdown by its parent or after a read iteration
		innerCtx, cancelFunc := context.WithCancel(ctx)
		fr.cancel = cancelFunc

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

				// Attempt to reconnect under some conditions
				if isNetworkError(err) {
					fr.logger.Info().Log("attempt to reconnect to stream subscription")
					if err := fr.reconnect(); err != nil {
						fr.logger.LogErrorf("unable to reconnect stream subscription: %v", err)
					}
				}
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
	sub.HandleFunc("/pipeline/{isolatedDirectory}/file-uploaded", fr.manuallyProduceFileUploaded())
}

type processableMessage struct {
	ctx context.Context
	msg *pubsub.Message
}

// handleMessage will listen for an incoming.ACHFile to pass off to an aggregator for the shard
// responsible. It does so with a database lookup and the fixed set of Shards from the file config.
func (fr *FileReceiver) handleMessage(ctx context.Context, sub stream.Subscription) chan error {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	out := make(chan error, 1)
	if sub == nil {
		return out
	}
	cleanup := func() {
		out <- nil
	}
	go func() {
		receiver := make(chan *processableMessage)
		go func() {
			msg, err := sub.Receive(ctx)

			traceCtx, span := telemetry.StartSpan(context.Background(), "file-receiver-handle-message")
			defer span.End()

			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				if strings.Contains(err.Error(), "Subscription has been Shutdown") {
					return
				}

				// Include the error in the span since it's interesting
				telemetry.RecordError(traceCtx, err)

				// Bubble up some errors to alerting
				if isNetworkError(err) {
					out <- err
					return
				}
				fr.logger.LogErrorf("ERROR receiving message: %v", err)
			}
			receiver <- &processableMessage{
				ctx: traceCtx,
				msg: msg,
			}
		}()

		select {
		case m := <-receiver:
			if m != nil && m.msg != nil {
				out <- fr.processMessage(m.ctx, m.msg)
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

func isNetworkError(err error) bool {
	return contains(err, "connect: ", "write:", "broken pipe", "pubsub", "EOF")
}

func contains(err error, options ...string) bool {
	es := err.Error()
	for i := range options {
		if strings.Contains(es, options[i]) {
			return true
		}
	}
	return false
}

func (fr *FileReceiver) processMessage(ctx context.Context, msg *pubsub.Message) error {
	// AutoCommit is a setting which will acknowledge messages with the pubsub service
	// immediately after receiving it. With this disabled messages are committed after
	// successful processing.
	//
	// Uncommitted messages will be redelivered and reprocessed, which can delay or
	// pause processing.
	committed := fr.shouldAutocommit()
	if committed {
		msg.Ack()
	}

	data := msg.Body
	var err error
	logger := fr.msgWrappedLogger(msg)

	// Optionally decode and decrypt message
	data, err = compliance.Reveal(fr.transformConfig, data)
	if err != nil {
		return logger.Error().LogErrorf("unable to reveal event: %v", err).Err()
	}

	event, readErr := models.Read(data)
	if event == nil {
		logger.Error().Log("no event read from data")
		return nil
	}
	if readErr != nil {
		logger.Error().LogErrorf("unable to read %s event: %v", event.Type, readErr)
	}

	var file *incoming.ACHFile
	switch evt := event.Event.(type) {
	case *models.QueueACHFile:
		f := incoming.ACHFile(*evt)
		file = &f

	case incoming.ACHFile:
		file = &evt

	case *models.CancelACHFile:
		err = fr.cancelACHFile(ctx, evt)
		if err != nil {
			return logger.With(log.Fields{
				"type": log.String(fmt.Sprintf("%T", evt)),
			}).LogError(err).Err()
		}
		if !committed {
			msg.Ack()
		}
		return nil
	}

	logger = logger.With(log.Fields{
		"type": log.String(fmt.Sprintf("%T", file)),
	})

	if file != nil {
		// Quit after we failed to read the event's file
		if readErr != nil {
			producerErr := fr.produceInvalidQueueFile(ctx, logger, *file, readErr)
			if err != nil {
				return logger.LogErrorf("problem producing InvalidQueueFile: %w", producerErr).Err()
			}
			return readErr
		} else {
			// Process the event like normal
			err = fr.processACHFile(ctx, *file)
			if err != nil {
				producerErr := fr.produceInvalidQueueFile(ctx, logger, *file, err)
				if err != nil {
					return logger.LogErrorf("problem producing InvalidQueueFile after processing file: %w", producerErr).Err()
				}
				return err
			}

			if !committed {
				msg.Ack()
			}

			return nil
		}
	}

	// Unhandled Message
	return logger.Error().LogError(errors.New("unhandled message")).Err()
}

func (fr *FileReceiver) shouldAutocommit() bool {
	kafkaConfig := fr.cfg.Inbound.Kafka
	if kafkaConfig == nil {
		return false
	}
	return kafkaConfig.AutoCommit
}

func (fr *FileReceiver) msgWrappedLogger(msg *pubsub.Message) log.Logger {
	logger := fr.logger.With(log.Fields{
		"loggableID": log.String(msg.LoggableID),
		"length":     log.Int(len(msg.Body)),
	})

	var details *sarama.ConsumerMessage
	ok := msg.As(details)
	if ok && details != nil {
		logger = logger.With(log.Fields{
			"key":       log.String(string(details.Key)),
			"topic":     log.String(details.Topic),
			"partition": log.Int64(int64(details.Partition)),
			"offset":    log.Int64(details.Offset),
		})
	}

	return logger
}

func (fr *FileReceiver) produceInvalidQueueFile(ctx context.Context, logger log.Logger, file incoming.ACHFile, err error) error {
	logger.Info().Log("producing InvalidQueueFile")

	hostname, _ := os.Hostname()

	return fr.eventEmitter.Send(ctx, models.Event{
		Event: models.InvalidQueueFile{
			File:     models.QueueACHFile(file),
			Error:    err.Error(),
			Hostname: hostname,
		},
	})
}

func (fr *FileReceiver) getAggregator(ctx context.Context, shardKey string) *aggregator {
	shardName, err := fr.shardRepository.Lookup(shardKey)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			fr.logger.Error().LogErrorf("problem looking up shardKey=%s: %v", shardKey, err)
			return nil
		}
	}
	if shardName == "" {
		// Often we have deployments with "SD-live-odfi" that have shard keys of "SD-$uuid".
		// We want "SD-live-odfi" (as a shard key) to represent "SD-live-odfi" (as a shard name).
		shardName = shardKey
	}

	agg, exists := fr.shardAggregators[shardName]
	if !exists {
		fr.logger.Warn().With(log.Fields{
			"shard_key":  log.String(shardKey),
			"shard_name": log.String(shardName),
		}).Log("found no shard so using default shard")

		telemetry.RecordError(ctx, fmt.Errorf("shardKey=%s shardName%s not found", shardKey, shardName))

		agg, exists = fr.shardAggregators[fr.defaultShardName]
		if !exists {
			filesMissingShardAggregators.With("shard", shardName).Add(1)
			fr.logger.Error().LogErrorf("no default shard %s configured after secondary lookup shardKey=%s shardName=%s", fr.defaultShardName, shardKey, shardName)
			return nil
		}
	}
	if agg == nil {
		fr.logger.Error().LogErrorf("nil shardAggregator for shardKey=%s shardName=%s", shardKey, shardName)
		return nil
	}
	return agg
}

func (fr *FileReceiver) processACHFile(ctx context.Context, file incoming.ACHFile) error {
	if file.FileID == "" || file.ShardKey == "" {
		return errors.New("missing fileID or shardKey")
	}

	err := file.Validate()
	if err != nil {
		telemetry.RecordError(ctx, err)
		fr.logger.Error().LogErrorf("invalid ACHFile: %v", err)
		return nil
	}

	// Pull Aggregator from the config
	agg := fr.getAggregator(ctx, file.ShardKey)
	if agg == nil {
		return fmt.Errorf("no aggregator for shard key %s found", file.ShardKey)
	}

	logger := fr.logger.With(log.Fields{
		"fileID":    log.String(file.FileID),
		"shardName": log.String(agg.shard.Name),
		"shardKey":  log.String(file.ShardKey),
	})

	// We only want to handle files once, so become the winner by saving the record.
	acceptanceData := files.AcceptedFile{
		FileID:     file.FileID,
		ShardKey:   file.ShardKey,
		AcceptedAt: time.Now().In(time.UTC),
	}
	acceptanceData.Hostname, _ = os.Hostname()

	fileRecordErr := fr.fileRepository.Record(ctx, acceptanceData)
	if fileRecordErr != nil {
		if database.UniqueViolation(fileRecordErr) {
			logger.Debug().Log("already handled file -- skipping")
			return nil
		}
		return logger.Error().LogErrorf("not handling received ACH file: %v", fileRecordErr).Err()
	}

	acceptFileResponse, acceptFileErr := agg.acceptFile(ctx, file)
	if acceptFileErr != nil {
		// Delete the record from files table
		deleteErr := fr.fileRepository.Cleanup(ctx, acceptanceData)
		if deleteErr != nil {
			logger.Error().LogErrorf("unable to cleanup files table: %v", err)
		}
		return logger.Error().LogErrorf("problem accepting file: %v", err).Err()
	}

	fr.QueueFileResponses <- acceptFileResponse

	// Record the file as accepted
	pendingFiles.With("shard", agg.shard.Name).Add(1)

	logger.Log("accepted ACH file")

	return nil
}

func (fr *FileReceiver) cancelACHFile(ctx context.Context, cancel *models.CancelACHFile) error {
	if cancel == nil || cancel.FileID == "" || cancel.ShardKey == "" {
		return errors.New("missing fileID or shardKey")
	}

	// Get the Aggregator from the config
	agg := fr.getAggregator(ctx, cancel.ShardKey)
	if agg == nil {
		return nil
	}

	logger := fr.logger.With(log.Fields{
		"fileID":    log.String(cancel.FileID),
		"shardName": log.String(agg.shard.Name),
		"shardKey":  log.String(cancel.ShardKey),
	})

	// Record the cancellation
	err := fr.fileRepository.Cancel(ctx, cancel.FileID)
	if err != nil {
		return logger.Error().LogErrorf("problem recording cancellation: %v", err).Err()
	}
	logger.Log("begin canceling ACH file")

	evt := incoming.CancelACHFile(*cancel)
	response, err := agg.cancelFile(ctx, evt)
	if err != nil {
		return logger.Error().LogErrorf("problem canceling file: %v", err).Err()
	}

	fr.CancellationResponses <- response

	logger.Log("finished cancel of file")
	return nil
}
