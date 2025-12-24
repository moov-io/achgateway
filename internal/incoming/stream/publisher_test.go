// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package stream

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/docker"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
	"gocloud.dev/pubsub"
)

func TestStream(t *testing.T) {
	ctx := context.Background()
	cfg := &service.Config{
		Inbound: service.Inbound{
			InMem: &service.InMemory{
				URL: "mem://moov",
			},
		},
	}

	topic, err := Topic(log.NewTestLogger(), cfg)
	require.NoError(t, err)
	defer topic.Shutdown(ctx)

	sub, err := OpenSubscription(log.NewTestLogger(), cfg)
	require.NoError(t, err)
	defer sub.Shutdown(ctx)

	// quick send and receive
	send(t, ctx, topic, "hello, world")
	if msg, err := receive(ctx, sub); err == nil {
		if msg != "hello, world" {
			t.Errorf("got %q", msg)
		}
	} else {
		t.Fatal(err)
	}
}

func TestStreamErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("-short flag enabled")
	}
	if !docker.Enabled() {
		t.Skip("Docker not enabled")
	}

	cfg := &service.Config{
		Inbound: service.Inbound{
			Kafka: &service.KafkaConfig{
				Brokers: []string{"localhost:19092"},
				Key:     "",
				Secret:  "",
				Topic:   "test1",
				TLS:     false,
			},
		},
	}
	ctx := context.Background()
	logger := log.NewTestLogger()

	topic, err := Topic(logger, cfg)
	require.NoError(t, err)
	defer topic.Shutdown(ctx)

	// Produce a message that's too big
	msg := &pubsub.Message{
		Body:     []byte(strings.Repeat("A", 1e9)),
		Metadata: make(map[string]string),
	}
	err = topic.Send(ctx, msg)
	require.ErrorContains(t, err, "Attempt to produce message larger than configured Producer.MaxMessageBytes")
}

func send(t *testing.T, ctx context.Context, topic Publisher, body string) *pubsub.Message {
	t.Helper()

	msg := &pubsub.Message{
		Body:     []byte(body),
		Metadata: make(map[string]string),
	}
	err := topic.Send(ctx, msg)
	if err != nil {
		t.Error(err)
	}
	return msg
}

func receive(ctx context.Context, sub Subscription) (string, error) {
	msg, err := sub.Receive(ctx)
	if err != nil {
		return "", err
	}
	if msg == nil {
		return "", errors.New("nil Message received")
	}
	msg.Ack()
	return string(msg.Body), nil
}
