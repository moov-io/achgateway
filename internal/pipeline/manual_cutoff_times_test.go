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
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestManualCutoffs_filter(t *testing.T) {
	var reqNames []string
	cfgName := "testing"

	require.False(t, exists(reqNames, cfgName))

	reqNames = append(reqNames, "live-odfi")
	require.False(t, exists(reqNames, cfgName))

	reqNames = append(reqNames, "testing")
	require.True(t, exists(reqNames, cfgName))
}

func TestFileReceiver__ManualCutoff(t *testing.T) {
	fr, wg := setupFileReceiver(t, nil) // waiter returns 'nil' error

	router := mux.NewRouter()
	router.Path("/trigger-cutoff").HandlerFunc(fr.triggerManualCutoff())

	var buf bytes.Buffer
	buf.WriteString(`{"shardNames":["testing"]}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/trigger-cutoff", &buf)
	router.ServeHTTP(w, req)

	wg.Wait()
	w.Flush()
	require.Equal(t, http.StatusOK, w.Code)

	var resp shardResponses
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	require.Len(t, resp.Shards, 1)
	require.Nil(t, resp.Shards["testing"])
}

func TestFileReceiver__ManualCutoff_RequestedShard_NotConfigured(t *testing.T) {
	fr, wg := setupFileReceiver(t, nil) // waiter returns 'nil' error

	router := mux.NewRouter()
	router.Path("/trigger-cutoff").HandlerFunc(fr.triggerManualCutoff())

	var buf bytes.Buffer
	buf.WriteString(`{"shardNames":["testing", "mordor"]}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/trigger-cutoff", &buf)
	router.ServeHTTP(w, req)

	wg.Wait()
	w.Flush()
	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp shardResponses
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	require.Len(t, resp.Shards, 2)
	require.Nil(t, resp.Shards["testing"])
	require.NotNil(t, resp.Shards["mordor"])
}

func TestFileReceiver__ManualCutoffErr(t *testing.T) {
	bad := errors.New("bad thing")
	fr, wg := setupFileReceiver(t, bad) // waiter returns error

	router := mux.NewRouter()
	router.Path("/trigger-cutoff").HandlerFunc(fr.triggerManualCutoff())

	var buf bytes.Buffer
	buf.WriteString(`{"shardNames":["testing"]}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/trigger-cutoff", &buf)
	router.ServeHTTP(w, req)

	wg.Wait()
	w.Flush()
	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp shardResponses
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	require.Len(t, resp.Shards, 1)
	require.Equal(t, "bad thing", *resp.Shards["testing"])
}

func setupFileReceiver(t *testing.T, waiterResponse error) (*FileReceiver, *sync.WaitGroup) {
	t.Helper()

	fr := &FileReceiver{
		logger:           log.NewTestLogger(),
		defaultShardName: "testing",
		shardAggregators: make(map[string]*aggregator),
	}

	cutoffTrigger := make(chan manuallyTriggeredCutoff)
	fr.shardAggregators["testing"] = &aggregator{
		shard: service.Shard{
			Name: "testing",
		},
		merger:        &MockXferMerging{},
		cutoffTrigger: cutoffTrigger,
	}

	fr.shardAggregators["ND-live-veridian"] = &aggregator{
		shard: service.Shard{
			Name: "ND-live-veridian",
		},
		merger:        &MockXferMerging{},
		cutoffTrigger: cutoffTrigger,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		// block for our waiter
		waiter := <-cutoffTrigger
		// mock out XferMerging's processing
		waiter.C <- waiterResponse
		// let the main test continue
		wg.Done()
	}()

	return fr, &wg
}
