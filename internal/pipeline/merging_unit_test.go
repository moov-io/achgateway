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
	"context"
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestMerging_createMergeConditions(t *testing.T) {
	m := &filesystemMerging{
		logger: log.NewTestLogger(),
	}

	conditions := m.createMergeConditions()
	require.Equal(t, int64(9_999_999_999_99), conditions.MaxDollarAmount)
	require.Equal(t, int64(ach.NachaFileDebitCreditLimit), conditions.MaxDollarAmount)

	// Set a lower amount limit in the achgateway config
	m.shard.Mergable.Conditions = &ach.Conditions{
		MaxLines:        100_000,
		MaxDollarAmount: 50_000_000_00,
	}
	conditions = m.createMergeConditions()
	require.Equal(t, 100_000, conditions.MaxLines)
	require.Equal(t, int64(50_000_000_00), conditions.MaxDollarAmount)

	// Don't allow the config to overflow what's allowed
	m.shard.Mergable.Conditions = &ach.Conditions{
		MaxLines:        100_000,
		MaxDollarAmount: 250_000_000_000_00,
	}
	conditions = m.createMergeConditions()
	require.Equal(t, 100_000, conditions.MaxLines)
	require.Equal(t, int64(ach.NachaFileDebitCreditLimit), conditions.MaxDollarAmount)
}

func TestMakeKey_StripsBatchNumber(t *testing.T) {
	bh1 := mockBatchPPDHeader()
	bh1.BatchNumber = 1
	bh2 := mockBatchPPDHeader()
	bh2.BatchNumber = 99

	entry := mockPPDEntryDetail()
	k1 := makeKey(bh1, entry)
	k2 := makeKey(bh2, entry)
	require.Equal(t, k1, k2, "keys must ignore batch number so merge renumbering still matches")
	require.NotContains(t, k1, "0000099")
}

func TestMakeKey_ShortHeader(t *testing.T) {
	entry := mockPPDEntryDetail()
	// Nil headers / entries should not panic or collide with a real key.
	require.NotEmpty(t, makeKey(nil, entry))
	nilKey := makeKey(nil, nil)
	require.NotEmpty(t, nilKey)
	require.NotEqual(t, makeKey(nil, entry), nilKey)
}

func TestMerging_HandleCancel_MissingFile(t *testing.T) {
	dir := t.TempDir()
	fsys, err := storage.NewFilesystem(dir)
	require.NoError(t, err)

	m := &filesystemMerging{
		logger: log.NewTestLogger(),
		shard: service.Shard{
			Name: "testing",
		},
		storage: fsys,
	}

	// Missing original: Successful must be false. Storage may return nil or an
	// error depending on ReplaceFile semantics for absent paths; either is OK
	// as long as the cancel is not reported successful.
	resp, _ := m.HandleCancel(context.Background(), incoming.CancelACHFile{
		FileID:   "does-not-exist.ach",
		ShardKey: "testing",
	})
	require.False(t, resp.Successful, "cancel of missing file must not report success")
}

func TestMerging_HandleCancel_AlreadyCanceled(t *testing.T) {
	dir := t.TempDir()
	fsys, err := storage.NewFilesystem(dir)
	require.NoError(t, err)

	m := &filesystemMerging{
		logger: log.NewTestLogger(),
		shard: service.Shard{
			Name: "testing",
		},
		storage: fsys,
	}

	// Queue then cancel once so the original exists; a second cancel is idempotent.
	require.NoError(t, m.storage.MkdirAll("mergable/testing"))
	require.NoError(t, m.storage.WriteFile("mergable/testing/pre.ach", []byte("101 test")))

	first, err := m.HandleCancel(context.Background(), incoming.CancelACHFile{
		FileID:   "pre.ach",
		ShardKey: "testing",
	})
	require.NoError(t, err)
	require.True(t, first.Successful, "first cancel of a queued file must succeed")

	second, err := m.HandleCancel(context.Background(), incoming.CancelACHFile{
		FileID:   "pre.ach",
		ShardKey: "testing",
	})
	require.NoError(t, err)
	require.True(t, second.Successful, "re-cancel after marker exists is idempotent")
}

func TestMerging_isolateMergableDir(t *testing.T) {
	dir := t.TempDir()
	fsys, err := storage.NewFilesystem(dir)
	require.NoError(t, err)

	m := &filesystemMerging{
		logger: log.NewTestLogger(),
		shard: service.Shard{
			Name: "shardA",
		},
		storage: fsys,
	}

	require.NoError(t, m.storage.MkdirAll("mergable/shardA"))
	require.NoError(t, m.storage.WriteFile("mergable/shardA/one.ach", []byte("101 test")))

	isolated, err := m.isolateMergableDir(context.Background())
	require.NoError(t, err)
	require.Contains(t, isolated, "shardA-")

	// Isolated dir should contain the file
	fd, err := m.storage.Open(filepath.Join(isolated, "one.ach"))
	require.NoError(t, err)
	if fd != nil {
		fd.Close()
	}
}

func TestMerging_HandleXfer_WithValidateOpts(t *testing.T) {
	merging, _ := testMergingSetup(t, service.Shard{Name: "SD-opts"})

	file := ach.NewFile()
	file.SetHeader(staticFileHeader())
	file.SetValidation(&ach.ValidateOpts{CustomReturnCodes: true})
	batch := ach.NewBatchPPD(mockBatchPPDHeader())
	entry := mockPPDEntryDetail()
	entry.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 9)
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	file.AddBatch(batch)
	require.NoError(t, file.Create())

	err := merging.HandleXfer(context.Background(), incoming.ACHFile{
		FileID:   "opts.ach",
		ShardKey: "SD-opts",
		File:     file,
	})
	require.NoError(t, err)

	// Sidecar validate opts file should exist next to the ACH
	m := merging.(*filesystemMerging)
	matches, err := m.storage.Glob("mergable/SD-opts/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, matches, "ValidateOpts sidecar should be written")
}
