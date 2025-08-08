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

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/files"
	"github.com/moov-io/achgateway/internal/incoming/stream/streamtest"
	"github.com/moov-io/achgateway/internal/incoming/web"
	"github.com/moov-io/achgateway/internal/pipeline"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestCancelFileAPI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	ctx := context.Background()
	logger := log.NewTestLogger()

	dir := t.TempDir()
	conf := &service.Config{
		Sharding: service.Sharding{
			Shards: []service.Shard{
				{
					Name: "testing",
					Cutoffs: service.Cutoffs{
						Timezone: "America/New_York",
						Windows:  []string{"12:00"},
					},
					UploadAgent: "mock",
				},
			},
			Default: "testing",
		},
		Upload: service.UploadAgents{
			Agents: []service.UploadAgent{
				{
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

	httpFilesTopic, httpFilesSub := streamtest.InmemStream(t)

	shardRepo := shards.NewInMemoryRepository()
	shardRepo.Add(service.ShardMapping{ShardKey: "testing", ShardName: "testing"}, database.NopInTx)

	fileRepo := &files.MockRepository{}

	fileReceiver, err := pipeline.Start(ctx, logger, conf, shardRepo, fileRepo, httpFilesSub)
	require.NoError(t, err)

	controller := web.NewFilesController(logger, service.HTTPConfig{}, httpFilesTopic, fileReceiver.QueueFileResponses, fileReceiver.CancellationResponses)
	r := mux.NewRouter()
	controller.AppendRoutes(r)

	// Accept file from stream
	fileID := base.ID()

	fd, err := os.Open(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)
	t.Cleanup(func() { fd.Close() })

	req := httptest.NewRequest("POST", fmt.Sprintf("/shards/SD-testing/files/%s.ach", fileID), fd)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify the file is written
	where := filepath.Join(dir, "mergable", "testing", "*.ach")
	require.Eventually(t, func() bool {
		filenames, err := filepath.Glob(where)
		require.NoError(t, err)

		parent, _ := filepath.Split(where)
		return slices.Contains(filenames, filepath.Join(parent, fileID+".ach"))
	}, 10*time.Second, 1*time.Second)

	// Now cancel that file
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/shards/SD-testing/files/%s.ach", fileID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response models.FileCancellationResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, fileID, response.FileID)
	require.Equal(t, "SD-testing", response.ShardKey)
	require.True(t, response.Successful)

	// Cancel it again (which should be successful)
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/shards/SD-testing/files/%s.ach", fileID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response2 models.FileCancellationResponse
	err = json.NewDecoder(w.Body).Decode(&response2)
	require.NoError(t, err)
	require.Equal(t, fileID, response2.FileID)
	require.Equal(t, "SD-testing", response2.ShardKey)
	require.True(t, response2.Successful)
}
