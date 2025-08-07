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
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/files"
	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/incoming/stream/streamtest"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gocloud.dev/pubsub"
)

type TestFileReceiver struct {
	*FileReceiver

	MergingDir string
	Publisher  stream.Publisher
	Events     *events.MockEmitter

	shardRepo shards.Repository
}

func (fr *TestFileReceiver) TriggerCutoff(t *testing.T) {
	t.Helper()

	agg, exists := fr.shardAggregators["testing"]
	if !exists {
		t.Fatal("testing shard not found")
	}

	waiter := manuallyTriggeredCutoff{
		C: make(chan error, 1),
	}
	agg.cutoffTrigger <- waiter
}

func testFileReceiver(t *testing.T) *TestFileReceiver {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test via -short")
	}

	ctx := context.Background()
	logger := log.NewTestLogger()

	dir := t.TempDir()
	conf := &service.Config{
		Inbound: service.Inbound{
			InMem: &service.InMemory{
				URL: "mem://" + t.Name(),
			},
		},
		Sharding: service.Sharding{
			Shards: []service.Shard{
				{
					Name: "testing",
					Cutoffs: service.Cutoffs{
						Timezone: "America/New_York",
						Windows:  []string{"12:00"},
					},
					OutboundFilenameTemplate: `{{ .ShardName }}-{{ date "150405.00000" }}.ach`,
					UploadAgent:              "mock",
				},
				{
					Name: "SD-live-odfi",
					Cutoffs: service.Cutoffs{
						Timezone: "America/Chicago",
						Windows:  []string{"5:00"},
					},
					OutboundFilenameTemplate: `{{ .ShardName }}-{{ date "150405.00000" }}.ach`,
					UploadAgent:              "mock",
				},
			},
			Default: "testing",
		},
		Upload: service.UploadAgents{
			Agents: []service.UploadAgent{
				{
					ID:   "mock",
					Mock: &service.MockAgent{},
				},
			},
			Merging: service.Merging{
				Storage: storage.Config{
					Filesystem: storage.FilesystemConfig{
						Directory: dir,
					},
				},
			},
		},
	}

	shardRepo := shards.NewInMemoryRepository()
	shardRepo.Add(service.ShardMapping{ShardKey: "testing", ShardName: "testing"}, database.NopInTx)

	fileRepo := &files.MockRepository{}

	filesTopic, _ := streamtest.InmemStream(t)
	fileReceiver, err := Start(ctx, logger, conf, shardRepo, fileRepo, nil)
	require.NoError(t, err)
	t.Cleanup(func() { fileReceiver.Shutdown() })

	var eventEmitter *events.MockEmitter
	if emitter, ok := fileReceiver.eventEmitter.(*events.MockEmitter); ok {
		eventEmitter = emitter
	}

	return &TestFileReceiver{
		FileReceiver: fileReceiver,
		MergingDir:   dir,
		Publisher:    filesTopic,
		Events:       eventEmitter,
		shardRepo:    shardRepo,
	}
}

func TestFileReceiver__InvalidQueueFile(t *testing.T) {
	fr := testFileReceiver(t)

	file, err := ach.ReadFile(filepath.Join("..", "incoming", "odfi", "testdata", "return-no-batch-controls.ach"))
	require.ErrorContains(t, err, ach.ErrFileHeader.Error())

	bs, err := compliance.Protect(nil, models.Event{
		Event: models.QueueACHFile{
			FileID:   base.ID(),
			ShardKey: "testing",
			File:     file,
		},
	})
	require.NoError(t, err)

	err = fr.Publisher.Send(context.Background(), &pubsub.Message{
		Body: bs,
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return len(fr.Events.Sent()) >= 1
	}, 1*time.Second, 100*time.Millisecond)

	event := fr.Events.Sent()[0]
	require.Equal(t, "InvalidQueueFile", event.Type)
	require.Equal(t, "*models.InvalidQueueFile", fmt.Sprintf("%T", event.Event))

	iqf, ok := event.Event.(*models.InvalidQueueFile)
	require.True(t, ok)
	require.Contains(t, iqf.Error, "reading QueueACHFile failed: ImmediateDestination")
}

func TestFileReceiver__getAggregator(t *testing.T) {
	fr := testFileReceiver(t)

	ctx := context.Background()
	agg := fr.getAggregator(ctx, "testing")
	require.NotNil(t, agg)

	mapping := service.ShardMapping{
		ShardKey:  "SD-" + uuid.NewString(),
		ShardName: "SD-live-odfi",
	}
	err := fr.shardRepo.Add(mapping, database.NopInTx)
	require.NoError(t, err)

	foundKey := fr.getAggregator(ctx, mapping.ShardKey)
	require.Equal(t, "SD-live-odfi", foundKey.shard.Name)

	foundName := fr.getAggregator(ctx, mapping.ShardName)
	require.Equal(t, foundKey, foundName)
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

func TestFileReceiver_AcceptFileErr(t *testing.T) {
	fr := testFileReceiver(t)

	m, ok := fr.shardAggregators["testing"].merger.(*filesystemMerging)
	require.True(t, ok)

	ms := &MockStorage{
		WriteFileErr: errors.New("bad thing"),
	}
	m.storage = ms

	// queue a file
	file, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)

	bs, err := compliance.Protect(nil, models.Event{
		Event: models.QueueACHFile{
			FileID:   base.ID(),
			ShardKey: "testing",
			File:     file,
		},
	})
	require.NoError(t, err)

	err = fr.Publisher.Send(context.Background(), &pubsub.Message{
		Body: bs,
	})
	require.NoError(t, err)

	// We should see an error, but can clear ms.WriteFileErr and retry
	// TODO(adam):
}
