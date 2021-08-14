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
	"testing"

	"github.com/moov-io/achgateway/internal/incoming/stream/streamtest"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"
)

func TestFileReceiver(t *testing.T) {
	testFileReceiver(t)
}

func testFileReceiver(t *testing.T) *FileReceiver {
	logger := log.NewNopLogger()
	shard := "testing"
	shardRepo := shards.NewMockRepository()
	shardAggregators := make(map[string]*aggregator)
	_, httpFiles := streamtest.InmemStream(t)
	_, streamFiles := streamtest.InmemStream(t)
	cfg := &models.TransformConfig{}

	fileRec := newFileReceiver(logger, shard, shardRepo, shardAggregators, httpFiles, streamFiles, cfg)
	fileRec.Start(context.Background())
	t.Cleanup(func() { fileRec.Shutdown() })

	return fileRec
}
