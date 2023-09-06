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

	cfg := service.ODFIReconciliation{
		Enabled:     true,
		PathMatcher: "recon",

		ProduceFileEvents:  true,
		ProduceEntryEvents: false,
	}

	file, _ := ach.ReadFile(filepath.Join("testdata", "recon.ach"))
	require.NotNil(t, file)

	// Add a bunch of batches and entries
	ed := file.Batches[0].GetEntries()[0]
	require.NotNil(t, ed)

	// Add batches and entries to file
	for i := 2; i < 500; i++ {
		entry := *ed
		entry.SetTraceNumber("32327427", i)

		// Keep batches to a certain size
		batch := file.Batches[len(file.Batches)-1]
		if i%25 == 0 {
			bh := batch.GetHeader()
			// Create a new batch
			batch, _ = ach.NewBatch(bh)
			file.AddBatch(batch)
		}
		batch.AddEntry(&entry)
	}
	require.Equal(t, 20, len(file.Batches))

	// Set ValidateOpts similar to what Processor sets
	file.SetValidation(&ach.ValidateOpts{
		AllowMissingFileControl:    true,
		AllowMissingFileHeader:     true,
		AllowUnorderedBatchNumbers: true,
	})

	// Re-build file
	for i := range file.Batches {
		require.NoError(t, file.Batches[i].Create())
	}
	require.NoError(t, file.Create())

	reconFile := File{
		Filepath: "recon.ach",
		ACHFile:  file,
	}

	t.Run("file events", func(t *testing.T) {
		eventsService := &events.MockEmitter{}
		emitter := CreditReconciliationEmitter(cfg, eventsService)
		require.NotNil(t, emitter)

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

	t.Run("entry events", func(t *testing.T) {
		cfg.ProduceFileEvents = false
		cfg.ProduceEntryEvents = true

		eventsService := &events.MockEmitter{}
		emitter := CreditReconciliationEmitter(cfg, eventsService)
		require.NotNil(t, emitter)

		err := emitter.Handle(logger, reconFile)
		require.NoError(t, err)

		var sent []models.Event
		require.Eventually(t, func() bool {
			sent = eventsService.Sent()
			return len(sent) > 0
		}, 5*time.Second, 100*time.Millisecond)

		require.Equal(t, 499, len(sent)) // one from previous subtest

		foundTraces := make(map[string]bool)
		for i := range sent {
			require.Equal(t, "ReconciliationEntry", sent[i].Type)

			event, ok := sent[i].Event.(*models.ReconciliationEntry)
			require.True(t, ok)
			foundTraces[event.Entry.TraceNumber] = true
		}
		require.Equal(t, 499, len(foundTraces)) // 499 unique trace numbers
	})
}
