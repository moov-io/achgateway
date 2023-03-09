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
	"errors"
	"testing"

	"github.com/moov-io/achgateway/internal/incoming/stream/streamtest"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestFileReceiver(t *testing.T) {
	testFileReceiver(t)
}

func testFileReceiver(t *testing.T) *FileReceiver {
	logger := log.NewNopLogger()
	conf := &service.Config{
		Sharding: service.Sharding{
			Default: "testing",
		},
	}
	shardRepo := shards.NewInMemoryRepository()
	shardAggregators := make(map[string]*aggregator)
	_, httpFiles := streamtest.InmemStream(t)
	cfg := &models.TransformConfig{}

	fileRec, err := newFileReceiver(logger, conf, shardRepo, shardAggregators, httpFiles, cfg)
	require.NoError(t, err)

	go fileRec.Start(context.Background())
	t.Cleanup(func() { fileRec.Shutdown() })

	return fileRec
}

func TestFileReceiver__shouldAutocommit(t *testing.T) {
	fr := testFileReceiver(t)

	// Ensure the setup is as we expect
	require.Nil(t, fr.cfg.Inbound.Kafka)
	require.False(t, fr.shouldAutocommit())

	// Set a config with AutoCommit disabled
	fr.cfg.Inbound.Kafka = &service.KafkaConfig{
		AutoCommit: false,
	}
	require.False(t, fr.shouldAutocommit())

	// Set .AutoCommit to true
	fr.cfg.Inbound.Kafka.AutoCommit = true
	require.True(t, fr.shouldAutocommit())
}

func TestFileReceiver__contains(t *testing.T) {
	err := errors.New("pubsub (code=Unknown): write tcp 10.100.53.92:45360->12.132.211.32:2222: write: broken pipe")

	require.True(t, contains(err, "write: "))
	require.True(t, contains(err, "pubsub"))

	require.False(t, contains(err, "connect: "))
	require.False(t, contains(err, "EOF"))
}
