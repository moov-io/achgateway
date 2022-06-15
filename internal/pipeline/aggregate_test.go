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
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestAggregateACHFile(t *testing.T) {
	shard := service.Shard{
		Name: "test",
		Cutoffs: service.Cutoffs{
			Timezone: "America/Los_Angeles",
			Windows:  []string{"10:30"},
		},
		UploadAgent: "ftp-live",
	}
	uploadAgents := service.UploadAgents{
		Agents: []service.UploadAgent{
			{
				ID:   "ftp-live",
				Mock: &service.MockAgent{},
				Paths: service.UploadPaths{
					Outbound: "/outbound",
				},
			},
		},
		DefaultAgentID: "ftp-live",
	}
	var errorAlerting service.AlertingConfig

	xfagg, err := newAggregator(log.NewNopLogger(), nil, &events.MockEmitter{}, shard, uploadAgents, errorAlerting)
	require.NoError(t, err)

	merge := &MockXferMerging{}
	xfagg.merger = merge

	go xfagg.Start(context.Background())

	// pass along a file
	file, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)

	err = xfagg.acceptFile(incoming.ACHFile{
		FileID:   "ppd-file1",
		ShardKey: "test",
		File:     file,
	})
	require.NoError(t, err)

	require.NotNil(t, merge.LatestFile)
	require.Equal(t, "ppd-file1", merge.LatestFile.FileID)
}

func TestAggregate_notifyAfterUpload(t *testing.T) {
	mockAgent := &upload.MockAgent{}

	shard := service.Shard{
		Name: "test",
		Cutoffs: service.Cutoffs{
			Timezone: "America/Los_Angeles",
			Windows:  []string{"10:30"},
		},
		UploadAgent: "mock-agent",
	}
	uploadAgents := service.UploadAgents{
		Agents: []service.UploadAgent{
			{
				ID:   "mock-agent",
				Mock: &service.MockAgent{},
				Paths: service.UploadPaths{
					Outbound: "/outbound",
				},
			},
		},
		DefaultAgentID: "mock-agent",
	}
	var errorAlerting service.AlertingConfig

	xfagg, err := newAggregator(log.NewNopLogger(), nil, &events.MockEmitter{}, shard, uploadAgents, errorAlerting)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		err := xfagg.notifyAfterUpload("filename.txt", nil, mockAgent, nil)
		require.NoError(t, err)
	})
}

func TestAggregate_notifyAfterUploadErr(t *testing.T) {
	mockAgent := &upload.MockAgent{}

	shard := service.Shard{
		Name: "test",
		Cutoffs: service.Cutoffs{
			Timezone: "America/Los_Angeles",
			Windows:  []string{"10:30"},
		},
		UploadAgent: "mock-agent",
	}
	uploadAgents := service.UploadAgents{
		Agents: []service.UploadAgent{
			{
				ID:   "mock-agent",
				Mock: &service.MockAgent{},
				Paths: service.UploadPaths{
					Outbound: "/outbound",
				},
			},
		},
		DefaultAgentID: "mock-agent",
	}
	var errorAlerting service.AlertingConfig

	xfagg, err := newAggregator(log.NewNopLogger(), nil, &events.MockEmitter{}, shard, uploadAgents, errorAlerting)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		err := xfagg.notifyAfterUpload("filename.txt", nil, mockAgent, errors.New("upload failed"))
		require.NoError(t, err)
	})
}
