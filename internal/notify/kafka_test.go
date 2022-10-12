// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestKafka(t *testing.T) {
	logger := log.NewTestLogger()
	conf := &service.KafkaConfig{
		Brokers:    []string{"127.0.0.1:9093"},
		Key:        "admin",
		Secret:     "secret",
		Topic:      "notify.kafka.v1",
		TLS:        false,
		AutoCommit: true,
		Transform: &models.TransformConfig{
			Encoding: &models.EncodingConfig{
				Base64: true,
			},
		},
	}
	kf, err := NewKafka(logger, conf)
	require.NoError(t, err)

	err = kf.Info(&Message{
		Direction: Upload,
		Filename:  "foo2.ach",
		File:      ach.NewFile(),
		Hostname:  "bank:22",
		Contents:  "hello, world",
	})
	require.NoError(t, err)

	// Setup a consumer
	conf.Group = fmt.Sprintf("achgateway-%d", time.Now().Unix())
	sub, err := stream.Subscription(logger, &service.Config{
		Inbound: service.Inbound{
			Kafka: conf,
		},
	})
	require.NoError(t, err)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	msg, err := sub.Receive(ctx)
	require.NoError(t, err)

	t.Log(string(msg.Body))
}
