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

package stream

import (
	"context"

	"github.com/moov-io/achgateway/internal/kafka"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
)

func Subscription(logger log.Logger, cfg *service.Config) (*pubsub.Subscription, error) {
	if cfg.Inbound.InMem != nil {
		sub, err := pubsub.OpenSubscription(context.Background(), cfg.Inbound.InMem.URL)
		if err != nil {
			return nil, err
		}
		logger.Info().Logf("setup %T inmem subscription", sub)
		return sub, nil
	}
	if cfg.Inbound.Kafka != nil {
		sub, err := kafka.OpenSubscription(logger, cfg.Inbound.Kafka)
		if err != nil {
			return nil, err
		}
		logger.Info().Logf("setup %T kafka subscription", sub)
		return sub, nil
	}
	return nil, nil
}
