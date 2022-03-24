// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

// Package stream exposes gocloud.dev/pubsub and side-loads various packages
// to register implementations such as kafka or in-memory. Please refer to
// specific documentation for each implementation.
//
//  - https://gocloud.dev/howto/pubsub/publish/
//  - https://gocloud.dev/howto/pubsub/subscribe/
//
// This package is designed as one import to bring in extra dependencies without
// requiring multiple projects to know what imports are needed.
package stream

import (
	"context"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

func Topic(logger log.Logger, cfg *service.Config) (*pubsub.Topic, error) {
	if cfg.Inbound.Kafka != nil {
		return createKafkaTopic(logger, cfg.Inbound.Kafka)
	}
	if cfg.Inbound.InMem != nil {
		return pubsub.OpenTopic(context.Background(), cfg.Inbound.InMem.URL)
	}
	return nil, nil
}

func createKafkaTopic(logger log.Logger, cfg *service.KafkaConfig) (*pubsub.Topic, error) {
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
		Log("setting up kafka topic")

	return kafkapubsub.OpenTopic(cfg.Brokers, config, cfg.Topic, nil)
}
