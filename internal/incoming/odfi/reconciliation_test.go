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
	"path/filepath"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestCreditReconciliation(t *testing.T) {
	logger := log.NewTestLogger()
	eventsService := &events.MockEmitter{}

	cfg := service.ODFIReconciliation{
		Enabled:     true,
		PathMatcher: "recon",
	}
	emitter := CreditReconciliationEmitter(cfg, eventsService)
	require.NotNil(t, emitter)

	file, _ := ach.ReadFile(filepath.Join("testdata", "recon.ach"))
	require.NotNil(t, file)

	// Set ValidateOpts similar to what Processor sets
	file.SetValidation(&ach.ValidateOpts{
		AllowMissingFileControl:    true,
		AllowMissingFileHeader:     true,
		AllowUnorderedBatchNumbers: true,
	})

	reconFile := File{
		Filepath: "recon.ach",
		ACHFile:  file,
	}

	t.Run("no conditions", func(t *testing.T) {
		err := emitter.Handle(logger, reconFile)
		require.NoError(t, err)

		var sent []models.Event
		require.Eventually(t, func() bool {
			sent = eventsService.Sent()
			return len(sent) > 0
		}, 5*time.Second, 100*time.Millisecond)
		require.Len(t, sent, 1)

		for i := range sent {
			require.Equal(t, "ReconciliationFile", sent[i].Type)
		}
	})

	t.Run("with conditions", func(t *testing.T) {
		cfg.Conditions = &ach.Conditions{
			MaxLines: 100,
		}

		err := emitter.Handle(logger, reconFile)
		require.NoError(t, err)

		var sent []models.Event
		require.Eventually(t, func() bool {
			sent = eventsService.Sent()
			return len(sent) > 0
		}, 5*time.Second, 100*time.Millisecond)
		require.Len(t, sent, 2) // one from previous subtest

		for i := range sent {
			require.Equal(t, "ReconciliationFile", sent[i].Type)
		}
	})
}
