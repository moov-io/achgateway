// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

// Package stream exposes gocloud.dev/pubsub and side-loads various packages
// to register implementations such as kafka or in-memory. Please refer to
// specific documentation for each implementation.
//
//   - https://gocloud.dev/howto/pubsub/publish/
//   - https://gocloud.dev/howto/pubsub/subscribe/
//
// This package is designed as one import to bring in extra dependencies without
// requiring multiple projects to know what imports are needed.
package stream

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Shopify/sarama"
	"github.com/moov-io/achgateway/internal/kafka"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

type Publisher interface {
	Send(ctx context.Context, m *pubsub.Message) error
	Shutdown(ctx context.Context) error
}

func Topic(logger log.Logger, cfg *service.Config) (Publisher, error) {
	if cfg.Inbound.InMem != nil {
		// Strip away any query params. They're only supported by subscriptions
		u, err := url.Parse(cfg.Inbound.InMem.URL)
		if err != nil {
			return nil, fmt.Errorf("parsing inbound in-mem url: %v", err)
		}

		addr := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		return pubsub.OpenTopic(context.Background(), addr)
	}
	if cfg.Inbound.Kafka != nil {
		topic, err := kafka.OpenTopic(logger, cfg.Inbound.Kafka)
		if err != nil {
			return nil, fmt.Errorf("creating topic: %w", err)
		}
		return &kafkaProducer{topic: topic}, nil
	}
	return nil, nil
}

type kafkaProducer struct {
	topic *pubsub.Topic
}

func (kp *kafkaProducer) Send(ctx context.Context, m *pubsub.Message) error {
	err := kp.topic.Send(ctx, m)
	if err != nil {
		var producerError sarama.ProducerError
		if kp.topic.ErrorAs(err, &producerError) {
			return fmt.Errorf("producer error sending message: %w", producerError)
		}
		var producerErrors sarama.ProducerErrors
		if kp.topic.ErrorAs(err, &producerErrors) {
			return fmt.Errorf("producer errors sending message: %w", producerErrors)
		}
		return fmt.Errorf("error sending message: %w", err)
	}
	return nil
}

func (kp *kafkaProducer) Shutdown(ctx context.Context) error {
	return kp.topic.Shutdown(ctx)
}
