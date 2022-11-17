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
	"github.com/moov-io/achgateway/internal/awssqs"

	"github.com/moov-io/achgateway/internal/kafka"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

func Topic(logger log.Logger, cfg *service.Config) (*pubsub.Topic, error) {
	if cfg.Inbound.InMem != nil {
		return pubsub.OpenTopic(context.Background(), cfg.Inbound.InMem.URL)
	}
	if cfg.Inbound.Kafka != nil {
		return kafka.OpenTopic(logger, cfg.Inbound.Kafka)
	}
	if cfg.Inbound.SQS != nil {
		return awssqs.OpenTopic(logger, cfg.Inbound.SQS)
	}
	return nil, nil
}
