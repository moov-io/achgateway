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

package odfi

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestIncoming(t *testing.T) {
	cfg := service.ODFIIncoming{
		Enabled: true,
	}
	recon := service.ODFIReconciliation{
		Enabled: true,
	}

	var output bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		output.Reset()
		defer r.Body.Close()

		bs, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		output.Write(bs)
		w.WriteHeader(http.StatusOK)
	}))

	logger := log.NewTestLogger()
	eventsService, err := events.NewEmitter(logger, &service.EventsConfig{
		Webhook: &service.WebhookConfig{
			Endpoint: server.URL + "/incoming",
		},
	})
	require.NoError(t, err)

	emitter := IncomingEmitter(cfg, recon, eventsService)
	require.NotNil(t, emitter)

	t.Run("no ACH file", func(t *testing.T) {
		err := emitter.Handle(logger, File{})
		require.NoError(t, err)
		require.Equal(t, "", output.String())
	})

	t.Run("no batch controls, missing creation date/time", func(t *testing.T) {
		bs, err := os.ReadFile(filepath.Join("testdata", "return-no-batch-controls.ach"))
		require.NoError(t, err)

		r := ach.NewReader(bytes.NewReader(bs))
		r.SetValidation(&ach.ValidateOpts{SkipAll: true})
		file, err := r.Read()
		require.ErrorContains(t, err, ach.ErrFileHeader.Error())

		// Clear out some FileHeader details
		file.Header.FileCreationDate = ""
		file.Header.FileCreationTime = ""

		err = emitter.Handle(logger, File{
			ACHFile: &file,
		})
		require.NoError(t, err)

		require.Contains(t, output.String(), `"individualName":"Best Co. #23          "`)
	})
}
