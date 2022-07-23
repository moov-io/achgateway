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

package rdfi

import (
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestScheduler(t *testing.T) {
	cfg := &service.Config{
		Logger: log.NewNopLogger(),
		Inbound: service.Inbound{
			ODFI: &service.ODFIFiles{
				Interval:   10 * time.Second,
				ShardNames: []string{"mock"},
				Storage: service.ODFIStorage{
					CleanupLocalDirectory: true,
					KeepRemoteFiles:       false,
					RemoveZeroByteFiles:   true,
				},
			},
		},
		Upload: service.UploadAgents{
			Agents: []service.UploadAgent{
				{
					ID:   "ftp-test",
					Mock: &service.MockAgent{},
				},
			},
			DefaultAgentID: "ftp-test",
		},
	}
	if testing.Verbose() {
		cfg.Logger = log.NewDefaultLogger()
	}

	processors := SetupProcessors(&MockProcessor{})
	schd, err := NewPeriodicScheduler(cfg.Logger, cfg, nil, processors)
	require.NoError(t, err)
	require.NotNil(t, schd)

	ss, ok := schd.(*PeriodicScheduler)
	if !ok {
		t.Fatalf("unexpected scheduler: %T", schd)
	}

	mock := &service.Shard{
		Name:        "mock",
		UploadAgent: "ftp-test",
	}
	if err := ss.tick(mock); err != nil {
		t.Fatal(err)
	}
}
