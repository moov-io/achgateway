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

package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/incoming/stream/streamtest"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestCreateFileHandler(t *testing.T) {
	topic, sub := streamtest.InmemStream(t)

	queueFileResponses := make(chan incoming.QueueACHFileResponse)
	cancellationResponses := make(chan models.FileCancellationResponse)
	controller := NewFilesController(log.NewTestLogger(), service.HTTPConfig{}, topic, queueFileResponses, cancellationResponses)
	r := mux.NewRouter()
	controller.AppendRoutes(r)

	// Setup the response
	go func() {
		time.Sleep(time.Second)

		queueFileResponses <- incoming.QueueACHFileResponse{
			FileID: "f1",
		}
	}()

	// Send a file over HTTP
	bs, _ := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "ppd-valid.json"))
	req := httptest.NewRequest("POST", "/shards/s1/files/f1", bytes.NewReader(bs))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req) // blocking call
	require.Equal(t, http.StatusOK, w.Code)

	// Wait for the subscription to receive the QueueACHFile
	msg, err := sub.Receive(context.Background())
	require.NoError(t, err)

	var file incoming.ACHFile
	err = models.ReadEvent(msg.Body, &file)
	require.NoError(t, err)

	// Verify the file details
	require.Equal(t, "f1", file.FileID)
	require.Equal(t, "s1", file.ShardKey)
	require.Equal(t, "231380104", file.File.Header.ImmediateDestination)

	validateOpts := file.File.GetValidation()
	require.NotNil(t, validateOpts)
	require.True(t, validateOpts.PreserveSpaces)
}

func TestCreateFileHandler_Error(t *testing.T) {
	topic, _ := streamtest.InmemStream(t)

	queueFileResponses := make(chan incoming.QueueACHFileResponse)
	cancellationResponses := make(chan models.FileCancellationResponse)
	controller := NewFilesController(log.NewTestLogger(), service.HTTPConfig{}, topic, queueFileResponses, cancellationResponses)
	r := mux.NewRouter()
	controller.AppendRoutes(r)

	// Setup the response
	go func() {
		time.Sleep(time.Second)

		queueFileResponses <- incoming.QueueACHFileResponse{
			FileID: "f1",
			Error:  "bad thing",
		}
	}()

	// Send a file over HTTP
	bs, _ := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "ppd-debit.ach"))
	req := httptest.NewRequest("POST", "/shards/s1/files/f1", bytes.NewReader(bs))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req) // blocking call
	require.Equal(t, http.StatusOK, w.Code)

	var resp models.QueueACHFileResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	require.Equal(t, "f1", resp.FileID)
	require.Equal(t, "", resp.ShardKey)
	require.Equal(t, "bad thing", resp.Error)
}

func TestCreateFileHandlerErr(t *testing.T) {
	topic, _ := streamtest.InmemStream(t)

	queueFileResponses := make(chan incoming.QueueACHFileResponse)
	cancellationResponses := make(chan models.FileCancellationResponse)
	controller := NewFilesController(log.NewTestLogger(), service.HTTPConfig{}, topic, queueFileResponses, cancellationResponses)
	r := mux.NewRouter()
	controller.AppendRoutes(r)

	// Send a file over HTTP
	req := httptest.NewRequest("POST", "/shards/s1/files/f1", strings.NewReader(`"invalid"`))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCancelFileHandler(t *testing.T) {
	topic, sub := streamtest.InmemStream(t)

	queueFileResponses := make(chan incoming.QueueACHFileResponse)
	cancellationResponses := make(chan models.FileCancellationResponse)
	controller := NewFilesController(log.NewTestLogger(), service.HTTPConfig{}, topic, queueFileResponses, cancellationResponses)
	r := mux.NewRouter()
	controller.AppendRoutes(r)

	// Cancel our file
	req := httptest.NewRequest("DELETE", "/shards/s2/files/f2.ach", nil)

	// Setup the response
	go func() {
		time.Sleep(time.Second)

		cancellationResponses <- models.FileCancellationResponse{
			FileID:     "f2.ach",
			ShardKey:   "s2",
			Successful: true,
		}
	}()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Verify our subscription receives a message
	msg, err := sub.Receive(context.Background())
	require.NoError(t, err)

	var file incoming.CancelACHFile
	require.NoError(t, models.ReadEvent(msg.Body, &file))

	require.Equal(t, "f2", file.FileID) // make sure .ach suffix is trimmed
	require.Equal(t, "s2", file.ShardKey)

	var response incoming.FileCancellationResponse
	json.NewDecoder(w.Body).Decode(&response)

	require.Equal(t, "f2.ach", response.FileID)
	require.Equal(t, "s2", response.ShardKey)
	require.True(t, response.Successful)
}
