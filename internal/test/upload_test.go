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
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/admintest"
	"github.com/moov-io/achgateway/internal/files"
	"github.com/moov-io/achgateway/internal/incoming/stream"
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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

var (
	uploadConf = &service.Config{
		Database: database.DatabaseConfig{
			DatabaseName: "achgateway",
			MySQL: &database.MySQLConfig{
				Address:  "tcp(127.0.0.1:3306)",
				User:     "root",
				Password: "root",
			},
		},
		Inbound: service.Inbound{
			InMem: &service.InMemory{
				URL: "mem://upload-test?ackdeadline=1s",
			},
		},
		Sharding: service.Sharding{
			Shards: []service.Shard{
				{
					Name: "prod",
					Cutoffs: service.Cutoffs{
						Timezone: "America/Los_Angeles",
						Windows:  []string{"12:03"},
					},
					OutboundFilenameTemplate: `{{ .ShardName }}-{{ date "150405.00000" }}-{{ .RoutingNumber }}.ach`,
					UploadAgent:              "ftp-live",

					Mergable: service.MergableConfig{
						MergeInGroupsOf: 100,
					},
				},
				{
					Name: "beta",
					Cutoffs: service.Cutoffs{
						Timezone: "America/New_York",
						Windows:  []string{"14:30"},
					},
					OutboundFilenameTemplate: `{{ .ShardName }}-{{ date "150405.00000" }}-{{ .RoutingNumber }}.ach`,
					UploadAgent:              "ftp-live",
					Mergable: service.MergableConfig{
						MergeInGroupsOf: 100,
					},
				},
				// This shard is never used, but we include it to verify merge/upload works
				{
					Name: "testing",
					Cutoffs: service.Cutoffs{
						Timezone: "America/New_York",
						Windows:  []string{"14:30"},
					},
					UploadAgent: "ftp-live",
					Mergable: service.MergableConfig{
						MergeInGroupsOf: 100,
					},
				},
			},
		},
		Upload: service.UploadAgents{
			Agents: []service.UploadAgent{
				{
					ID: "ftp-live",
					FTP: &service.FTP{
						Hostname: "127.0.0.1:2121",
						Username: "admin",
						Password: "123456",
					},
					Paths: service.UploadPaths{
						Inbound:        "inbound",
						Outbound:       "<todo>",
						Reconciliation: "reconciliation",
						Return:         "returned",
					},
				},
			},
			Merging: service.Merging{
				Storage: storage.Config{
					Encryption: storage.EncryptionConfig{
						AES: &storage.AESConfig{
							Base64Key: mergingAESKey,
						},
						Encoding: "base64",
					},
				},
			},
		},
	}

	mergingAESKey = base64.RawStdEncoding.EncodeToString(bytes.Repeat([]byte("1"), 32))
)

func TestUploads(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test via -short")
	}

	// Force a clean mergable directory for the test
	require.NoError(t, os.RemoveAll("storage"))

	ctx := context.Background()
	logger := log.NewTestLogger()

	shardRepo := shards.NewInMemoryRepository()
	shardKeys := setupShards(t, shardRepo)

	httpPub, httpSub := streamtest.InmemStream(t)
	streamTopic, err := stream.Topic(logger, uploadConf)
	require.NoError(t, err)
	defer streamTopic.Shutdown(context.Background())

	outboundPath := setupTestDirectory(t, uploadConf)
	fileRepo := &files.MockRepository{}

	fileReceiver, err := pipeline.Start(ctx, logger, uploadConf, shardRepo, fileRepo, httpSub)
	require.NoError(t, err)
	t.Cleanup(func() { fileReceiver.Shutdown() })

	fileController := web.NewFilesController(logger, service.HTTPConfig{}, httpPub, fileReceiver.CancellationResponses)
	r := mux.NewRouter()
	fileController.AppendRoutes(r)

	adminServer := admintest.Server(t)
	fileReceiver.RegisterAdminRoutes(adminServer)

	// Force the stream subscription to fail
	flakeySub := streamtest.FailingSubscription(errors.New("write: broken pipe"))
	fileReceiver.ReplaceStreamFiles(flakeySub)
	require.Contains(t, fmt.Sprintf("%#v", fileReceiver), "streamFiles:(*streamtest.FailedSubscription)")

	// Upload our files
	createdEntries := 0
	canceledEntries := 0
	erroredSubscriptions := 0
	var createdFileIDs, canceledFileIDs []string

	iterations := 500
	var g errgroup.Group
	g.SetLimit(iterations / 100)

	for i := 0; i < iterations; i++ {
		shardKey := shardKeys[i%10]
		fileID := uuid.NewString()
		file := randomACHFile(t)
		createdEntries += countEntries(file)

		g.Go(func() error {
			w := submitFile(t, r, shardKey, fileID, file)
			require.Equal(t, http.StatusOK, w.Code)
			return nil
		})

		canceled := maybeCancelFile(t, r, shardKey, fileID, file)
		if canceled > 0 {
			canceledEntries += canceled
			canceledFileIDs = append(canceledFileIDs, fileID)
		} else {
			createdFileIDs = append(createdFileIDs, fileID)
		}

		// Force the subscription to fail sometimes
		if err := causeSubscriptionFailure(t); err != nil {
			flakeySub := streamtest.FailingSubscription(err)
			fileReceiver.ReplaceStreamFiles(flakeySub)
			erroredSubscriptions += 1
		}
	}
	require.NoError(t, g.Wait())

	t.Logf("created %d entries (in %d files) and canceled %d entries (in %d files)", createdEntries, len(createdFileIDs), canceledEntries, len(canceledFileIDs))
	require.Greater(t, createdEntries, 0, "created entries")
	require.Greater(t, canceledEntries, 0, "canceled entries")

	// Pause for long enough that all files get accepted
	wait := time.Duration(5*iterations) * time.Millisecond // 50k iterations is 4m10s
	if wait < 2*time.Minute {
		wait = 2 * time.Minute
	}
	tick := wait / 10
	require.Eventually(t, func() bool {
		// Count how many files are in mergable/beta and mergable/prod, which should be the created + canceled files.
		betaFDs, _ := os.ReadDir(filepath.Join("storage", "mergable", "beta"))
		prodFDs, _ := os.ReadDir(filepath.Join("storage", "mergable", "prod"))
		t.Logf("found %d beta and %d prod mergable files, expected %d + %d", len(betaFDs), len(prodFDs), len(createdFileIDs), len(canceledFileIDs))

		return (len(betaFDs) + len(prodFDs)) >= (len(createdFileIDs) + len(canceledFileIDs))
	}, wait, tick)

	// Manual upload of all files
	var buf bytes.Buffer
	buf.WriteString(`{"shardNames":["prod", "beta", "testing"]}`)
	req, _ := http.NewRequest("PUT", "http://"+adminServer.BindAddr()+"/trigger-cutoff", &buf)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	// Wait before verifying filesystem results
	time.Sleep(10 * time.Second)

	filenamePrefixCounts := countFilenamePrefixes(t, outboundPath)
	require.Greater(t, filenamePrefixCounts["BETA"], 0)
	require.Greater(t, filenamePrefixCounts["PROD"], 0)

	// Verify no files are left in mergable/
	mergableFiles, err := ach.ReadDir(filepath.Join("storage", "mergable"))
	require.NoError(t, err)
	require.Equal(t, 0, len(mergableFiles))

	// Verify each fileID was isolated on disk
	verifyFilesWereIsolated(t, createdFileIDs)

	uploadedFiles, err := ach.ReadDir(outboundPath)
	require.NoError(t, err)

	expected := createdEntries - canceledEntries
	found := countAllEntries(uploadedFiles)
	t.Logf("found %d entries of %d expected (%d canceled) (%d errored) from %d uploaded files", found, expected, canceledEntries, erroredSubscriptions, len(uploadedFiles))
	require.Equal(t, expected, found)
}

func setupTestDirectory(t *testing.T, cfg *service.Config) string {
	t.Helper()

	dir, err := os.MkdirTemp(filepath.Join("..", "..", "testdata", "ftp-server"), "outbound-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	cfg.Upload.Agents[0].Paths.Outbound = filepath.Base(dir)
	return dir
}

func countFilenamePrefixes(t *testing.T, outboundPath string) map[string]int {
	t.Helper()

	out := make(map[string]int)

	entries, err := os.ReadDir(outboundPath)
	require.NoError(t, err)

	for i := range entries {
		info, err := entries[i].Info()
		require.NoError(t, err)

		if info.Mode().IsRegular() {
			parts := strings.Split(filepath.Base(entries[i].Name()), "-")
			out[parts[0]] += 1
		}
	}
	require.Len(t, out, 2, "unexpected shard counts")
	t.Logf("found %v shards", out)
	return out
}

func countAllEntries(files []*ach.File) (out int) {
	for i := range files {
		out += countEntries(files[i])
	}
	return out
}

func countEntries(file *ach.File) (out int) {
	if file == nil {
		return 0
	}
	for i := range file.Batches {
		out += len(file.Batches[i].GetEntries())
	}
	return out
}

func randomInt(t *testing.T, max int64) int64 {
	t.Helper()

	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		t.Fatal(err)
	}
	return n.Int64()
}

func randomACHFile(t *testing.T) *ach.File {
	t.Helper()

	if randomInt(t, 100)%2 == 0 {
		file, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
		require.NoError(t, err)
		return randomTraceNumbers(t, file)
	}

	bs, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ppd-valid.json"))
	require.NoError(t, err)
	file, err := ach.FileFromJSON(bs)
	require.NoError(t, err)
	return randomTraceNumbers(t, file)
}

func traceNumber(t *testing.T, routingNumber string) string {
	t.Helper()

	num := randomInt(t, 1e15)
	v := fmt.Sprintf("%s%d", routingNumber, num)
	if utf8.RuneCountInString(v) > 15 {
		return v[:15]
	}
	return v
}

func randomTraceNumbers(t *testing.T, file *ach.File) *ach.File {
	t.Helper()

	for i := range file.Batches {
		b, err := ach.NewBatch(file.Batches[i].GetHeader())
		require.NoError(t, err)

		entries := file.Batches[i].GetEntries()
		for i := range entries {
			if i == 0 {
				entries[i].TraceNumber = traceNumber(t, entries[i].TraceNumber[:8])
				b.AddEntry(entries[i])
			} else {
				n, _ := strconv.Atoi(entries[0].TraceNumber)
				entries[i].TraceNumber = fmt.Sprintf("%d", n+1)
				b.AddEntry(entries[i])
			}
		}

		require.NoError(t, b.Create())
		file.Batches[i] = b
	}
	require.NoError(t, file.Create())
	return file
}

func setupShards(t *testing.T, repo *shards.InMemoryRepository) []string {
	t.Helper()

	var out []string
	for i := 0; i < 10; i++ {
		shardKey := base.ID()
		if i%2 == 0 {
			repo.Add(service.ShardMapping{ShardKey: shardKey, ShardName: "prod"}, database.NopInTx)
		} else {
			repo.Add(service.ShardMapping{ShardKey: shardKey, ShardName: "beta"}, database.NopInTx)
		}
		out = append(out, shardKey)
	}
	return out
}

func submitFile(t *testing.T, r *mux.Router, shardKey, fileID string, file *ach.File) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(file); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", fmt.Sprintf("/shards/%s/files/%s", shardKey, fileID), &body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func maybeCancelFile(t *testing.T, r *mux.Router, shardKey, fileID string, file *ach.File) int {
	t.Helper()

	if shouldCancelFile(t) {
		go cancelFile(t, r, shardKey, fileID)
		return countEntries(file)
	}

	return 0
}

func shouldCancelFile(t *testing.T) bool {
	t.Helper()

	return randomInt(t, 100) <= 5 // 5%
}

func cancelFile(t *testing.T, r *mux.Router, shardKey, fileID string) {
	t.Helper()

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(&models.CancelACHFile{
		ShardKey: shardKey,
		FileID:   fileID,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/shards/%s/files/%s", shardKey, fileID), &body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

var subscriptionFailures = []error{
	io.EOF,
	errors.New("write: broken pipe"),
	errors.New("contains: pubsub error"),
}

func causeSubscriptionFailure(t *testing.T) error {
	t.Helper()

	n := randomInt(t, 100)
	if n <= 5 { // 5%
		idx := (len(subscriptionFailures) - 1) % (int(n) + 1)
		return subscriptionFailures[idx]
	}
	return nil
}

func firstDirectory(t *testing.T, fsys fs.FS, prefix string) string {
	t.Helper()

	matches, err := fs.Glob(fsys, fmt.Sprintf("%s-*", prefix))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	return matches[0]
}

func verifyFilesWereIsolated(t *testing.T, fileIDs []string) {
	t.Helper()

	fsys := os.DirFS("storage")
	beta, prod := firstDirectory(t, fsys, "beta"), firstDirectory(t, fsys, "prod")

	for i := range fileIDs {
		betaMatches, _ := fs.Glob(fsys, filepath.Join(beta, fmt.Sprintf("%s.*", fileIDs[i])))
		prodMatches, _ := fs.Glob(fsys, filepath.Join(prod, fmt.Sprintf("%s.*", fileIDs[i])))

		total := len(betaMatches) + len(prodMatches)
		if total == 0 {
			t.Errorf("fileID[%d] %s not found in beta or prod shard", i, fileIDs[i])
		}
	}
}
