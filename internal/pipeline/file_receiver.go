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
	"fmt"

	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
)

// fileReceiver accepts an ACH file from a number of pubsub Subscriptions and
// finds the appropiate aggregator for the shardKey.
type fileReceiver struct {
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
) *fileReceiver {
	return &fileReceiver{
		logger:           logger,
		shardRepository:  shardRepository,
		shardAggregators: shardAggregators,
		httpFiles:        httpFiles,
		streamFiles:      streamFiles,
	}
}

func (fr *fileReceiver) Start(ctx context.Context) {
	for {
		select {
		case err := <-fr.handleMessage(ctx, fr.httpFiles): // TODO
			fmt.Printf("http err=%#v\n", err)

		case err := <-fr.handleMessage(ctx, fr.streamFiles): // TODO
			fmt.Printf("stream err=%#v\n", err)

		case <-ctx.Done():
			fr.Shutdown()
			return
		}
	}
}

func (fr *fileReceiver) Shutdown() {
	fr.logger.Log("shutting down xfer aggregation")

	if err := fr.httpFiles.Shutdown(context.Background()); err != nil {
		fr.logger.LogErrorf("problem shutting down http file subscription: %v", err)
	}
	if err := fr.streamFiles.Shutdown(context.Background()); err != nil {
		fr.logger.LogErrorf("problem shutting down stream file subscription: %v", err)
	}
}

func (fr *fileReceiver) handleMessage(ctx context.Context, sub *pubsub.Subscription) chan error {
	// 1. shardRepository.Lookup(shardKey string) (shardName, error)
	// 2. lookup fr.shardAggregators[shardName]
	// 3. call aggregator.acceptFile(*incoming.ACHFile)

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

			// fr.logger.Logf("begin handle received message of %d bytes", len(msg.Body))
			msg.Ack()

			if err := agg.acceptFile(file); err != nil {
				fr.logger.Error().LogErrorf("problem accepting file under shardName=%s", shardName)

				// TODO(adam): PD alert, notify people

				out <- err
			} else {
				// log about successful message handling
				out <- nil
			}
		} else {
			fr.logger.Log("nil message received")
		}
	}()
	return out
}
