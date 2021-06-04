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
	"encoding/json"

	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
)

// FileReceiver accepts an ACH file from a number of pubsub Subscriptions and
// finds the appropiate aggregator for the shardKey.
type FileReceiver struct {
	logger log.Logger

	shardRepository  shards.Repository
	shardAggregators map[string]*aggregator

	httpFiles   *pubsub.Subscription
	streamFiles *pubsub.Subscription
}

func newFileReceiver(
	logger log.Logger,
	shardRepository shards.Repository,
	shardAggregators map[string]*aggregator,
	httpFiles *pubsub.Subscription,
	streamFiles *pubsub.Subscription,
) *FileReceiver {
	return &FileReceiver{
		logger:           logger,
		shardRepository:  shardRepository,
		shardAggregators: shardAggregators,
		httpFiles:        httpFiles,
		streamFiles:      streamFiles,
	}
}

func (fr *FileReceiver) Start(ctx context.Context) {
	for {
		select {
		case err := <-fr.handleMessage(ctx, fr.httpFiles):
			fr.logger.LogErrorf("error handling http file: %v", err)

		case err := <-fr.handleMessage(ctx, fr.streamFiles):
			fr.logger.LogErrorf("error handling stream file: %v", err)

		case <-ctx.Done():
			fr.Shutdown()
			return
		}
	}
}

func (fr *FileReceiver) Shutdown() {
	fr.logger.Log("shutting down xfer aggregation")

	if err := fr.httpFiles.Shutdown(context.Background()); err != nil {
		fr.logger.LogErrorf("problem shutting down http file subscription: %v", err)
	}
	if err := fr.streamFiles.Shutdown(context.Background()); err != nil {
		fr.logger.LogErrorf("problem shutting down stream file subscription: %v", err)
	}
}

// handleMessage will listen for an incoming.ACHFile to pass off to an aggregator for the shard
// responsible. It does so with a database lookup and the fixed set of Shards from the file config.
func (fr *FileReceiver) handleMessage(ctx context.Context, sub *pubsub.Subscription) chan error {
	out := make(chan error, 1)
	if sub == nil {
		return out
	}
	go func() {
		msg, err := sub.Receive(ctx)
		if err != nil {
			fr.logger.LogErrorf("ERROR receiving message: %v", err)
		}
		if msg != nil {
			var file incoming.ACHFile
			if err := json.Unmarshal(msg.Body, &file); err != nil {
				fr.logger.Error().LogErrorf("unable to parse incoming.ACHFile: %v", err)
				return
			}
			fr.logger.Logf("begin handle received ACHFile=%s of %d bytes", file.FileID, len(msg.Body))

			shardName, err := fr.shardRepository.Lookup(file.ShardKey)
			if err != nil {
				fr.logger.Error().LogErrorf("problem looking up shardKey=%s: %v", file.ShardKey, err)
				return
			}

			agg, exists := fr.shardAggregators[shardName]
			if !exists {
				fr.logger.Error().LogErrorf("missing shardAggregator for shardKey=%s shardName=%s", file.ShardKey, shardName)
				return
			}

			msg.Ack()

			if err := agg.acceptFile(file); err != nil {
				fr.logger.Error().LogErrorf("problem accepting file under shardName=%s", shardName)
				out <- err
			} else {
				fr.logger.Logf("finished handling ACHFile=%s", file.FileID)
				out <- nil
			}
		} else {
			fr.logger.Log("nil message received")
		}
	}()
	return out
}
