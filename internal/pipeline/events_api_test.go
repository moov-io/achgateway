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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/admintest"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"

	"github.com/stretchr/testify/require"
	"gocloud.dev/pubsub"
)

func TestEventsAPI_FileUploaded(t *testing.T) {
	adminServer := admintest.Server(t)

	fr := testFileReceiver(t)
	fr.RegisterAdminRoutes(adminServer)

	// Write a file that's produced
	agg, exists := fr.shardAggregators["testing"]
	require.True(t, exists)
	require.NotNil(t, agg)

	m, ok := agg.merger.(*filesystemMerging)
	require.True(t, ok)
	require.NotNil(t, m)

	file, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)

	fileID := base.ID()
	bs, err := compliance.Protect(nil, models.Event{
		Event: models.QueueACHFile{
			FileID:   fileID,
			ShardKey: "testing",
			File:     file,
		},
	})
	require.NoError(t, err)

	err = fr.Publisher.Send(context.Background(), &pubsub.Message{
		Body: bs,
	})
	require.NoError(t, err)

	// Verify the file is pending on disk
	require.Eventually(t, func() bool {
		found, _ := os.ReadDir(filepath.Join(fr.MergingDir, "mergable", "testing"))
		return len(found) > 0
	}, 5*time.Second, 100*time.Millisecond)

	// Isolate the directory (on upload)
	fr.TriggerCutoff(t)

	var found []os.DirEntry
	require.Eventually(t, func() bool {
		found, err = os.ReadDir(fr.MergingDir)
		require.NoError(t, err)
		return len(found) > 1
	}, 5*time.Second, 100*time.Millisecond)

	var uploadDir string
	for i := range found {
		if found[i].Name() != "mergable" {
			uploadDir = found[i].Name()
		}
	}
	require.NotEmpty(t, uploadDir)

	// Reproduce FileUploaded event
	address := fmt.Sprintf("http://%s/shards/testing/pipeline/%s/file-uploaded?filename=foo.ach", adminServer.BindAddr(), uploadDir)
	req, err := http.NewRequest("PUT", address, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify that FileUploaded was produced twice:
	//  - once for the manual trigger
	//  - once for the /file-uploaded endpoint
	emitter, ok := agg.eventEmitter.(*events.MockEmitter)
	require.True(t, ok)
	require.Eventually(t, func() bool {
		return len(emitter.Sent()) == 2
	}, 5*time.Second, 100*time.Millisecond)

	// Check fields of the FileUploaded events
	sentEvents := emitter.Sent()
	require.Len(t, sentEvents, 2)

	for i := range sentEvents {
		switch v := sentEvents[i].Event.(type) {
		case *models.FileUploaded:
			require.Equal(t, fileID, v.FileID)

			// The first event has the actual filename,
			// but the second event reads the query param
			switch i {
			case 0:
				// Example: TESTING-143425.52750.ach
				require.True(t, strings.HasPrefix(v.Filename, "TESTING-"), v.Filename)
				require.True(t, strings.HasSuffix(v.Filename, ".ach"), v.Filename)
			case 1:
				require.Equal(t, "foo.ach", v.Filename)
			}
		default:
			t.Errorf("unexpected %#v", v)
		}
	}
}

func TestEventsAPI_FileUploadedErrors(t *testing.T) {
	adminServer := admintest.Server(t)

	fr := testFileReceiver(t)
	fr.RegisterAdminRoutes(adminServer)

	t.Run("Call /file-uploaded on a shard that doesn't exist", func(t *testing.T) {
		address := fmt.Sprintf("http://%s/shards/missing/pipeline/missing-12345/file-uploaded", adminServer.BindAddr())
		req, err := http.NewRequest("PUT", address, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Call /file-uploaded on a directory that doesn't exist", func(t *testing.T) {
		address := fmt.Sprintf("http://%s/shards/testing/pipeline/missing/file-uploaded", adminServer.BindAddr())
		req, err := http.NewRequest("PUT", address, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Call /file-uploaded on insecure paths", func(t *testing.T) {
		paths := []string{"../../etc/passwd", "/etc/passwd"}
		for i := range paths {
			address := fmt.Sprintf("http://%s/shards/testing/pipeline/%s/file-uploaded", adminServer.BindAddr(), paths[i])
			req, err := http.NewRequest("PUT", address, nil)
			require.NoError(t, err, fmt.Sprintf("on address %s", address))
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusNotFound, resp.StatusCode, fmt.Sprintf("on address %s", address))
		}
	})

}
