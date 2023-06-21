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

package models

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/base"

	"github.com/stretchr/testify/require"
)

func TestEvent(t *testing.T) {
	check := func(t *testing.T, inner interface{}, matchers ...string) {
		t.Helper()
		evt := Event{
			Event: inner,
		}
		bs, err := json.Marshal(evt)
		require.NoError(t, err)
		for i := range matchers {
			require.Contains(t, string(bs), matchers[i])
		}
	}
	// Verfiy every event type
	check(t, CorrectionFile{
		File: ach.NewFile(),
	}, `"type":"CorrectionFile"`)

	check(t, IncomingFile{
		File: ach.NewFile(),
	}, `"type":"IncomingFile"`)

	check(t, ReturnFile{
		File: ach.NewFile(),
	}, `"type":"ReturnFile"`)

	check(t, FileUploaded{
		FileID:     base.ID(),
		ShardKey:   base.ID(),
		UploadedAt: time.Now(),
	}, `"type":"FileUploaded"`)

	check(t, InvalidQueueFile{
		File: QueueACHFile{
			File: ach.NewFile(),
		},
		Error: "missing abc123",
	}, `"type":"InvalidQueueFile"`, `"error":"missing abc123"`)
}

func TestRead(t *testing.T) {
	orig := CancelACHFile{
		FileID:   base.ID(),
		ShardKey: base.ID(),
	}

	bs := (Event{
		Event: orig,
	}).Bytes()

	second, err := Read(bs)
	require.NoError(t, err)
	require.Equal(t, "CancelACHFile", second.Type)

	cancel, ok := second.Event.(*CancelACHFile)
	require.True(t, ok)

	require.Equal(t, orig.FileID, cancel.FileID)
	require.Equal(t, orig.ShardKey, cancel.ShardKey)
}

func TestRead__InvalidQueueFile(t *testing.T) {
	data := []byte(`{"event":{"file":{"id":"07e45f053b513d7ccc64cdd4c16da93fb3f57ea8","shardKey":"testing","file":{"id":"","fileHeader":{"id":"","immediateDestination":"","immediateOrigin":"","fileCreationDate":"","fileCreationTime":"","fileIDModifier":"A","immediateDestinationName":"","immediateOriginName":""},"batches":null,"IATBatches":null,"fileControl":{"id":"","batchCount":0,"blockCount":0,"entryAddendaCount":0,"entryHash":0,"totalDebit":0,"totalCredit":0},"fileADVControl":{"id":"","batchCount":0,"entryAddendaCount":0,"entryHash":0,"totalDebit":0,"totalCredit":0},"NotificationOfChange":null,"ReturnEntries":null,"validateOpts":{"skipAll":false,"requireABAOrigin":false,"bypassOriginValidation":false,"bypassDestinationValidation":false,"customTraceNumbers":false,"allowZeroBatches":false,"allowMissingFileHeader":false,"allowMissingFileControl":false,"bypassCompanyIdentificationMatch":false,"customReturnCodes":false,"unequalServiceClassCode":false,"allowUnorderedBatchNumbers":false,"allowInvalidCheckDigit":false,"unequalAddendaCounts":false,"preserveSpaces":false,"allowInvalidAmounts":false}}},"error":"reading QueueACHFile failed: ImmediateDestination            is a mandatory field and has a default value, did you use the constructor?"},"type":"InvalidQueueFile"}`)

	t.Run("Read", func(t *testing.T) {
		found, err := Read(data)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, "InvalidQueueFile", found.Type)

		iqf, ok := found.Event.(*InvalidQueueFile)
		require.True(t, ok)
		require.Equal(t, "reading QueueACHFile failed: ImmediateDestination            is a mandatory field and has a default value, did you use the constructor?", iqf.Error)
	})

	t.Run("ReadEvent", func(t *testing.T) {
		iqf := &InvalidQueueFile{}
		iqf.SetValidation(&ach.ValidateOpts{
			SkipAll: true,
		})
		err := ReadEvent(data, iqf)
		require.NoError(t, err)
		require.Equal(t, "reading QueueACHFile failed: ImmediateDestination            is a mandatory field and has a default value, did you use the constructor?", iqf.Error)
	})
}

func TestPartialReconciliationFile(t *testing.T) {
	file, err := ach.ReadFile(filepath.Join("testdata", "partial-recon.ach"))
	require.NotNil(t, err)
	require.True(t, base.Has(err, ach.ErrFileHeader))

	var bs bytes.Buffer
	json.NewEncoder(&bs).Encode(Event{
		Event: &ReconciliationFile{
			Filename: "partial-recon.ach",
			File:     file,
		},
	})

	var evt ReconciliationFile
	evt.SetValidation(&ach.ValidateOpts{
		AllowMissingFileHeader:  true,
		AllowMissingFileControl: true,
	})

	err = ReadEvent(bs.Bytes(), &evt)
	require.NoError(t, err)

	batches := evt.File.Batches
	require.Len(t, batches, 2)
}
