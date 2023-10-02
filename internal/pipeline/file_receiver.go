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
	"fmt"
	"strings"
	"sync"

	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/log"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
)

// FileReceiver accepts an ACH file from a number of pubsub Subscriptions and
// finds the appropriate aggregator for the shardKey.
type FileReceiver struct {
	logger log.Logger
	cfg    *service.Config
	mu     sync.RWMutex

	defaultShardName string

	eventEmitter events.Emitter

	shardRepository  shards.Repository
	shardAggregators map[string]*aggregator

	httpFiles   stream.Subscription
	streamFiles stream.Subscription

	transformConfig *models.TransformConfig
}

func newFileReceiver(
	logger log.Logger,
	cfg *service.Config,
	eventEmitter events.Emitter,
	shardRepository shards.Repository,
	shardAggregators map[string]*aggregator,
	httpFiles stream.Subscription,
	transformConfig *models.TransformConfig,
) (*FileReceiver, error) {
	// Create FileReceiver and connect streamFiles
	fr := &FileReceiver{
		logger:           logger,
		cfg:              cfg,
		eventEmitter:     eventEmitter,
		defaultShardName: cfg.Sharding.Default,
		shardRepository:  shardRepository,
		shardAggregators: shardAggregators,
		httpFiles:        httpFiles,
		transformConfig:  transformConfig,
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

func (fr *FileReceiver) ReplaceStreamFiles(sub stream.Subscription) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	// Close an existing stream subscription
	if fr.streamFiles != nil {
		fr.streamFiles.Shutdown(context.Background())
	}
	fr.streamFiles = sub
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
		receiver := make(chan *pubsub.Message)
		go func() {
			msg, err := sub.Receive(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				if strings.Contains(err.Error(), "Subscription has been Shutdown") {
					return
				}
				// Bubble up some errors to alerting
				if isNetworkError(err) {
					out <- err
					return
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

func (fr *FileReceiver) processMessage(msg *pubsub.Message) error {
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
		logger.LogErrorf("unable to reveal event: %v", err)
		data = msg.Body
	}

	event, readErr := models.Read(data)
	if readErr != nil {
		logger.LogErrorf("unable to read %s event: %v", event.Type, readErr)
	}

	var file *incoming.ACHFile
	switch evt := event.Event.(type) {
	case *models.QueueACHFile:
		f := incoming.ACHFile(*evt)
		file = &f

	case incoming.ACHFile:
		file = &evt

	case *models.CancelACHFile:
		err = fr.cancelACHFile(evt)
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
			producerErr := fr.produceInvalidQueueFile(logger, *file, readErr)
			if err != nil {
				return logger.LogErrorf("problem producing InvalidQueueFile: %w", producerErr).Err()
			}
			return readErr
		} else {
			// Process the event like normal
			err = fr.processACHFile(*file)
			if err != nil {
				producerErr := fr.produceInvalidQueueFile(logger, *file, err)
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
	return logger.LogError(errors.New("unhandled message")).Err()
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

func (fr *FileReceiver) produceInvalidQueueFile(logger log.Logger, file incoming.ACHFile, err error) error {
	logger.Info().Log("producing InvalidQueueFile")

	return fr.eventEmitter.Send(models.Event{
		Event: models.InvalidQueueFile{
			File:  models.QueueACHFile(file),
			Error: err.Error(),
		},
	})
}

func (fr *FileReceiver) getAggregator(shardKey string) *aggregator {
	shardName, err := fr.shardRepository.Lookup(shardKey)
	if err != nil {
		fr.logger.Error().LogErrorf("problem looking up shardKey=%s: %v", shardKey, err)
		return nil
	}

	agg, exists := fr.shardAggregators[shardName]
	if !exists {
		fr.logger.Warn().With(log.Fields{
			"shard_key":  log.String(shardKey),
			"shard_name": log.String(shardName),
		}).Log("found no shard so using default shard")

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
		return fmt.Errorf("no aggregator for shard key %s found", file.ShardKey)
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
