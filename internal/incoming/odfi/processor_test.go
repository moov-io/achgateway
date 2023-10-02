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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/alerting"
	"github.com/moov-io/achgateway/internal/audittrail"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
	"gocloud.dev/pubsub"
)

func TestProcessor(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "invalid.ach"), []byte("invalid-ach-file"), 0600)
	require.NoError(t, err)

	proc := &MockProcessor{}
	processors := SetupProcessors(proc)
	auditSaver := &AuditSaver{
		storage:  &audittrail.MockStorage{},
		hostname: "ftp.foo.com",
	}
	logger := log.NewTestLogger()
	var validation ach.ValidateOpts
	alerters, _ := alerting.NewAlerters(service.ErrorAlerting{})

	// By reading a file without ACH FileHeaders we still want to try and process
	// Batches inside of it if any are found, so reading this kind of file shouldn't
	// return an error from reading the file.
	err = processDir(logger, dir, alerters, auditSaver, validation, processors)
	require.NoError(t, err)

	require.NotNil(t, proc.HandledFile)
	require.NotNil(t, proc.HandledFile.ACHFile)
	require.Equal(t, "7ffdca32898fc89e5e680d0a01e9e1c2a1cd2717", proc.HandledFile.ACHFile.ID)

	// Real world file
	path := filepath.Join("..", "..", "..", "testdata", "HMBRAD_ACHEXPORT_1001_08_19_2022_09_10")
	err = processFile(logger, path, alerters, auditSaver, validation, processors)
	if err != nil {
		require.ErrorContains(t, err, "record:FileHeader *ach.FieldError FileCreationDate  is a mandatory field")
	}

	// Verify saved file path
	ms, ok := auditSaver.storage.(*audittrail.MockStorage)
	require.True(t, ok)

	yyyymmdd := time.Now().Format("2006-01-02")
	require.Equal(t, fmt.Sprintf("odfi/ftp.foo.com/testdata/%s/HMBRAD_ACHEXPORT_1001_08_19_2022_09_10", yyyymmdd), ms.SavedFilepath)
}

func TestProcessor_populateHashes(t *testing.T) {
	file, err := ach.ReadFile(filepath.Join("testdata", "forward.ach"))
	require.ErrorContains(t, err, ach.ErrFileHeader.Error())

	populateHashes(file)
	require.Equal(t, "", file.Batches[0].ID())

	entries := file.Batches[0].GetEntries()
	require.Equal(t, "389723d3a8293a802169b5db27f288d32e96b9c6", entries[0].ID)
}

func TestProcessor_populateIatHashes(t *testing.T) {
	file, err := ach.ReadFile(filepath.Join("testdata", "iat-credit.ach"))
	require.NoError(t, err)

	populateHashes(file)
	require.Equal(t, "", file.IATBatches[0].ID)

	entries := file.IATBatches[0].GetEntries()
	require.Equal(t, "f26f52d2603771f52c983bf6062ba503fd126087", entries[0].ID)
}

func TestProcessor_MultiReturnCorrection(t *testing.T) {
	cfg := service.ODFIFiles{
		Processors: service.ODFIProcessors{
			Corrections: service.ODFICorrections{
				Enabled: true,
			},
			Returns: service.ODFIReturns{
				Enabled: true,
			},
		},
	}
	logger := log.NewTestLogger()
	eventsConfig := &service.EventsConfig{
		Stream: &service.EventsStream{
			InMem: &service.InMemory{
				URL: "mem://odfi-multi-ret-cor",
			},
		},
	}
	emitter, err := events.NewEmitter(logger, eventsConfig)
	require.NoError(t, err)

	processors := SetupProcessors(
		ReturnEmitter(cfg.Processors.Returns, emitter),
		CorrectionEmitter(cfg.Processors.Corrections, emitter),
	)

	file, err := ach.ReadFile(filepath.Join("testdata", "return-no-batch-controls.ach"))
	require.ErrorContains(t, err, ach.ErrFileHeader.Error())
	require.Len(t, file.Batches, 2)

	// Setup consumer prior to sending messages
	ctx := context.Background()
	sub, err := pubsub.OpenSubscription(ctx, eventsConfig.Stream.InMem.URL)
	require.NoError(t, err)

	// Process ACH file
	err = processors.HandleAll(logger, File{
		ACHFile: file,
	})
	require.NoError(t, err)

	// Consume events
	var correction *models.CorrectionFile
	var returned *models.ReturnFile
	for i := 0; i < 2; i++ {
		msg, err := sub.Receive(ctx)
		require.NoError(t, err)

		evt, _ := models.ReadWithOpts(msg.Body, &ach.ValidateOpts{
			AllowMissingFileHeader:  true,
			AllowMissingFileControl: true,
		})
		require.NotNil(t, evt)

		switch e := evt.Event.(type) {
		case *models.CorrectionFile:
			correction = e
		case *models.ReturnFile:
			returned = e
		}
	}

	require.NotNil(t, correction)
	require.Len(t, correction.Corrections, 1)
	cor := correction.Corrections[0]
	require.Equal(t, "121042882", cor.Header.CompanyIdentification)
	require.Len(t, cor.Entries, 1)
	require.NotNil(t, cor.Entries[0].Addenda98)
	require.Nil(t, cor.Entries[0].Addenda99)

	require.NotNil(t, returned)
	require.Len(t, returned.Returns, 1)
	ret := returned.Returns[0]
	require.Equal(t, "123456789", ret.Header.CompanyIdentification)
	require.Len(t, ret.Entries, 1)
	require.Nil(t, ret.Entries[0].Addenda98)
	require.NotNil(t, ret.Entries[0].Addenda99)
}
