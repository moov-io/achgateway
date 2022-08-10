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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
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
	"github.com/moov-io/achgateway/internal/consul"
	"github.com/moov-io/achgateway/internal/incoming/stream/streamtest"
	"github.com/moov-io/achgateway/internal/incoming/web"
	"github.com/moov-io/achgateway/internal/pipeline"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

var (
	cfg = &service.Config{
		Database: database.DatabaseConfig{
			DatabaseName: "achgateway",
			MySQL: &database.MySQLConfig{
				Address:  "tcp(127.0.0.1:3306)",
				User:     "root",
				Password: "root",
			},
		},
		Consul: &consul.Config{
			Address:     "http://127.0.0.1:8500",
			SessionPath: "achgateway/upload-test/",
		},
		Inbound: service.Inbound{
			InMem: &service.InMemory{},
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
				},
				{
					Name: "beta",
					Cutoffs: service.Cutoffs{
						Timezone: "America/New_York",
						Windows:  []string{"14:30"},
					},
					OutboundFilenameTemplate: `{{ .ShardName }}-{{ date "150405.00000" }}-{{ .RoutingNumber }}.ach`,
					UploadAgent:              "ftp-live",
				},
				// This shard is never used, but we include it to verify merge/upload works
				{
					Name: "testing",
					Cutoffs: service.Cutoffs{
						Timezone: "America/New_York",
						Windows:  []string{"14:30"},
					},
					UploadAgent: "ftp-live",
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

func init() {
	rand.Seed(time.Now().Unix())
}

func TestUploads(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test via -short")
	}

	ctx := context.Background()
	logger := log.NewDefaultLogger()

	consulClient, err := consul.NewConsulClient(logger, cfg.Consul)
	require.NoError(t, err)

	shardRepo := shards.NewMockRepository()
	shardKeys := setupShards(t, shardRepo)

	httpPub, httpSub := streamtest.InmemStream(t)
	_, streamSub := streamtest.InmemStream(t)

	fileController := web.NewFilesController(logger, service.HTTPConfig{}, httpPub)
	r := mux.NewRouter()
	fileController.AppendRoutes(r)

	outboundPath := setupTestDirectory(t, cfg)
	fileReceiver, err := pipeline.Start(ctx, logger, cfg, consulClient, shardRepo, httpSub, streamSub)
	require.NoError(t, err)
	t.Cleanup(func() { fileReceiver.Shutdown() })

	adminServer := admin.NewServer(":0")
	go adminServer.Listen()
	defer adminServer.Shutdown()
	fileReceiver.RegisterAdminRoutes(adminServer)

	// Upload our files
	createdEntries := 0
	canceledEntries := 0
	for i := 0; i < 1000; i++ {
		shardKey := shardKeys[i%10]
		fileID := base.ID()
		file := randomACHFile(t)
		createdEntries += countEntries(file)
		w := submitFile(t, r, shardKey, fileID, file)
		require.Equal(t, http.StatusOK, w.Code)

		canceledEntries += maybeCancelFile(t, r, shardKey, fileID, file)
	}

	t.Logf("created %d entries and canceled %d entries", createdEntries, canceledEntries)
	require.Greater(t, createdEntries, 0)
	require.Greater(t, canceledEntries, 0)
	time.Sleep(5 * time.Second)

	req, _ := http.NewRequest("PUT", "http://"+adminServer.BindAddr()+"/trigger-cutoff", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	filenamePrefixCounts := countFilenamePrefixes(t, outboundPath)
	require.Greater(t, filenamePrefixCounts["BETA"], 0)
	require.Greater(t, filenamePrefixCounts["PROD"], 0)

	createdFiles, err := ach.ReadDir(outboundPath)
	require.NoError(t, err)

	expected := createdEntries - canceledEntries
	found := countAllEntries(createdFiles)
	t.Logf("found %d entries of %d expected (%d canceled)", found, expected, canceledEntries)
	require.Equal(t, found, expected)
}

func setupTestDirectory(t *testing.T, cfg *service.Config) string {
	dir, err := os.MkdirTemp(filepath.Join("..", "..", "testdata", "ftp-server"), "upload-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	cfg.Upload.Agents[0].Paths.Outbound = filepath.Base(dir)
	return dir
}

func countFilenamePrefixes(t *testing.T, outboundPath string) map[string]int {
	t.Helper()

	out := make(map[string]int)

	entries, err := os.ReadDir(outboundPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range entries {
		info, err := entries[i].Info()
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().IsRegular() {
			parts := strings.Split(filepath.Base(entries[i].Name()), "-")
			out[parts[0]] += 1
		}
	}
	if len(out) != 2 {
		t.Fatalf("unexpected shard counts: %v", out)
	}
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

func randomACHFile(t *testing.T) *ach.File {
	//nolint:gosec
	if rand.Int31()%2 == 0 {
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

func traceNumber(routingNumber string) string {
	//nolint:gosec
	num := rand.Int63n(1e15)
	v := fmt.Sprintf("%s%d", routingNumber, num)
	if utf8.RuneCountInString(v) > 15 {
		return v[:15]
	}
	return v
}

func randomTraceNumbers(t *testing.T, file *ach.File) *ach.File {
	for i := range file.Batches {
		b, err := ach.NewBatch(file.Batches[i].GetHeader())
		require.NoError(t, err)

		entries := file.Batches[i].GetEntries()
		for i := range entries {
			if i == 0 {
				entries[i].TraceNumber = traceNumber(entries[i].TraceNumber[:8])
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

func setupShards(t *testing.T, repo *shards.MockRepository) []string {
	var out []string
	for i := 0; i < 10; i++ {
		shardKey := base.ID()
		if i%2 == 0 {
			repo.Shards[shardKey] = service.ShardMapping{ShardKey: shardKey, ShardName: "prod"}
		} else {
			repo.Shards[shardKey] = service.ShardMapping{ShardKey: shardKey, ShardName: "beta"}
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
		cancelFile(t, r, shardKey, fileID)
		return countEntries(file)
	}

	return 0
}

func shouldCancelFile(t *testing.T) bool {
	t.Helper()

	return rand.Int63n(100)%10 == 0 //nolint:gosec
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
