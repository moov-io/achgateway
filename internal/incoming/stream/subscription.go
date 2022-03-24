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
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
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
		sub, err := createKafkaSubscription(logger, cfg.Inbound.Kafka)
		if err != nil {
			return nil, err
		}
		logger.Info().Logf("setup %T kafka subscription", sub)
		return sub, nil
	}
	return nil, nil
}

func createKafkaSubscription(logger log.Logger, cfg *service.KafkaConfig) (*pubsub.Subscription, error) {
	config := kafkapubsub.MinimalConfig()
	config.Version = minKafkaVersion
	config.Net.TLS.Enable = cfg.TLS

	config.Net.SASL.Enable = cfg.Key != ""
	config.Net.SASL.Mechanism = sarama.SASLMechanism("PLAIN")
	config.Net.SASL.User = cfg.Key
	config.Net.SASL.Password = cfg.Secret

	// AutoCommit in Sarama refers to "automated publishing of consumer offsets
	// to the broker" rather than a Kafka broker's meaning of "commit consumer
	// offsets on read" which leads to "at-most-once" delivery.
	config.Consumer.Offsets.AutoCommit.Enable = cfg.AutoCommit

	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.IsolationLevel = sarama.ReadCommitted

	logger.Info().
		Set("tls", log.Bool(cfg.TLS)).
		Set("group", log.String(cfg.Group)).
		Set("sasl.enable", log.Bool(config.Net.SASL.Enable)).
		Set("sasl.user", log.String(cfg.Key)).
		Set("topic", log.String(cfg.Topic)).
		Log("setting up kafka subscription")

	return kafkapubsub.OpenSubscription(cfg.Brokers, config, cfg.Group, []string{cfg.Topic}, &kafkapubsub.SubscriptionOptions{
		WaitForJoin: 10 * time.Second,
	})
}
