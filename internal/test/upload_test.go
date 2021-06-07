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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/consul"
	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/incoming/web"
	"github.com/moov-io/achgateway/internal/pipeline"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/base"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"
	"github.com/stretchr/testify/require"
	"gocloud.dev/pubsub"
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
			Address:             "127.0.0.1:8500",
			Scheme:              "http",
			SessionPath:         "achgateway/upload-test/",
			HealthCheckInterval: 10 * time.Second,
		},
		Inbound: service.Inbound{
			InMem: &service.InMemory{},
		},
		Shards: service.Shards{
			{
				Name: "prod",
				Cutoffs: service.Cutoffs{
					Timezone: "America/Los_Angeles",
					Windows:  []string{"12:03"},
				},
				OutboundFilenameTemplate: `{{ date "20060102" }}-{{ date "150405.00000" }}-{{ .RoutingNumber }}.ach`,
				UploadAgent:              "ftp-test",
			},
		},
		Upload: service.UploadAgents{
			Agents: []service.UploadAgent{
				{
					ID: "ftp-test",
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
			DefaultAgentID: "ftp-test",
		},
	}
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
	wrapper := consul.NewWrapper(logger, consulClient)

	shardRepo := shards.NewMockRepository()
	shardKeys := setupShards(t, shardRepo)

	httpPub, httpSub := setupInmemStream(t)
	_, streamSub := setupInmemStream(t)

	fileController := web.NewFilesController(logger, httpPub)
	r := mux.NewRouter()
	fileController.AppendRoutes(r)

	outboundPath := setupTestDirectory(t, cfg)
	fileReceiver, err := pipeline.Start(ctx, logger, cfg, wrapper, shardRepo, httpSub, streamSub)
	require.NoError(t, err)
	t.Cleanup(func() { fileReceiver.Shutdown() })

	adminServer := admin.NewServer(":0")
	go adminServer.Listen()
	defer adminServer.Shutdown()
	fileReceiver.RegisterAdminRoutes(adminServer)

	// Upload our files
	createdEntries := 0
	for i := 0; i < 1000; i++ {
		shardKey := shardKeys[i%10]
		fileID := base.ID()
		file := randomACHFile(t)
		createdEntries += countEntries(file)
		w := submitFile(t, r, shardKey, fileID, file)
		require.Equal(t, http.StatusOK, w.Code)
	}

	t.Logf("created %d entries", createdEntries)
	time.Sleep(5 * time.Second)

	req, _ := http.NewRequest("PUT", "http://"+adminServer.BindAddr()+"/trigger-cutoff", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	createdFiles, err := ach.ReadDir(outboundPath)
	require.NoError(t, err)

	expected := countAllEntries(createdFiles)
	t.Logf("found %d entries of %d expected", expected, createdEntries)
	require.Equal(t, expected, createdEntries)
}

func setupTestDirectory(t *testing.T, cfg *service.Config) string {
	dir, err := ioutil.TempDir(filepath.Join("..", "..", "testdata", "ftp-server"), "upload-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	cfg.Upload.Agents[0].Paths.Outbound = filepath.Base(dir)
	return dir
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
	if rand.Int31()%2 == 0 {
		file, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
		require.NoError(t, err)
		return randomTraceNumbers(t, file)
	}

	bs, err := ioutil.ReadFile(filepath.Join("..", "..", "testdata", "ppd-valid.json"))
	require.NoError(t, err)
	file, err := ach.FileFromJSON(bs)
	require.NoError(t, err)
	return randomTraceNumbers(t, file)
}

func traceNumber(routingNumber string) string {
	v := fmt.Sprintf("%s%d", routingNumber, rand.Int63n(1e15))
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
		repo.Shards[shardKey] = "prod"
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

func setupInmemStream(t *testing.T) (*pubsub.Topic, *pubsub.Subscription) {
	t.Helper()

	conf := &service.Config{
		Inbound: service.Inbound{
			InMem: &service.InMemory{
				URL: fmt.Sprintf("mem://achgateway-%s", t.Name()),
			},
		},
	}
	topic, err := stream.Topic(log.NewNopLogger(), conf)
	require.NoError(t, err)

	sub, err := stream.Subscription(log.NewNopLogger(), conf)
	require.NoError(t, err)
	t.Cleanup(func() { sub.Shutdown(context.Background()) })

	return topic, sub
}
