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
	"fmt"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
)

func Start(
	ctx context.Context,
	logger log.Logger,
	cfg *service.Config,
	adminServer *admin.Server,
	shardRepository shards.Repository,
	httpFiles, streamFiles *pubsub.Subscription) (*FileReceiver, error) {

	// register each shard's aggregator
	shardAggregators := make(map[string]*aggregator)
	for i := range cfg.Shards {
		xfagg, err := newAggregator(logger, cfg.Shards[i], cfg.Upload)
		if err != nil {
			return nil, fmt.Errorf("problem starting shard=%s: %v", cfg.Shards[i].Name, err)
		}

		go xfagg.Start(ctx)

		shardAggregators[cfg.Shards[i].Name] = xfagg
	}

	// register our fileReceiver and start it
	receiver := newFileReceiver(logger, shardRepository, shardAggregators, httpFiles, streamFiles)
	go receiver.Start(ctx)

	return receiver, nil
}
