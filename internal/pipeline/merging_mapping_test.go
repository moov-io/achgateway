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
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestMissingMappedInputFiles(t *testing.T) {
	expected := []string{
		"isolated/a.ach",
		"b.ach",
		"isolated/c.ach",
		"a.ach", // duplicate path form of a
	}
	merged := mergedFiles{
		{InputFilepaths: []string{"a.ach", "c.ach"}},
	}
	require.Equal(t, []string{"b.ach"}, missingMappedInputFiles(expected, merged))
	require.Empty(t, missingMappedInputFiles(nil, merged))
	require.Empty(t, missingMappedInputFiles(expected, mergedFiles{
		{InputFilepaths: []string{"a.ach"}},
		{InputFilepaths: []string{"b.ach", "c.ach"}},
	}))
}

func TestSummarizeFilenames(t *testing.T) {
	require.Equal(t, "", summarizeFilenames(nil, 10))
	require.Equal(t, "a, b", summarizeFilenames([]string{"a", "b"}, 10))
	require.Equal(t, "a, b, ... (+2 more)", summarizeFilenames([]string{"a", "b", "c", "d"}, 2))
}

func TestBuildDirMapping_SkipsBadInputContinues(t *testing.T) {
	// One unreadable/invalid input must not prevent mapping the rest.
	dir := t.TempDir()
	fsys, err := storage.NewFilesystem(dir)
	require.NoError(t, err)

	m := &filesystemMerging{
		logger:  log.NewTestLogger(),
		storage: fsys,
	}

	// Valid ACH
	valid := ach.NewFile()
	valid.SetHeader(staticFileHeader())
	batch := ach.NewBatchPPD(mockBatchPPDHeader())
	entry := mockPPDEntryDetail()
	entry.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 7)
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	valid.AddBatch(batch)
	require.NoError(t, valid.Create())

	var buf bytes.Buffer
	require.NoError(t, ach.NewWriter(&buf).Write(valid))
	require.NoError(t, fsys.MkdirAll("iso"))
	require.NoError(t, fsys.WriteFile("iso/good.ach", buf.Bytes()))
	require.NoError(t, fsys.WriteFile("iso/bad.ach", []byte("not-an-ach-file")))

	mappings, errs := m.buildDirMapping("iso", nil)
	require.Len(t, errs, 1)
	require.ErrorContains(t, errs[0], "bad.ach")
	require.NotEmpty(t, mappings, "good.ach should still be mapped")

	// good.ach filename should appear in mapping values
	found := false
	for _, names := range mappings {
		if slices.Contains(names, "good.ach") {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestMerging_DuplicateEntryIdentity_DifferentFileIDs_SingleUploadBothMapped(t *testing.T) {
	// Same entry under different file_ids: one merged ODFI upload containing both
	// entries, and both file_ids mapped for FileUploaded.
	merging, _ := testMergingSetup(t, serviceShard("SD-dup"))

	file := ach.NewFile()
	file.SetHeader(staticFileHeader())
	batch := ach.NewBatchPPD(mockBatchPPDHeader())
	entry := mockPPDEntryDetail()
	entry.Amount = 12345
	entry.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 42)
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	file.AddBatch(batch)
	require.NoError(t, file.Create())

	for _, id := range []string{"dup-a.ach", "dup-b.ach"} {
		err := merging.HandleXfer(context.Background(), incoming.ACHFile{
			FileID:   id,
			ShardKey: "SD-dup",
			File:     file,
		})
		require.NoError(t, err)
	}

	var uploads int
	var uploadedFile *ach.File
	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, f *ach.File) (string, error) {
		uploads++
		uploadedFile = f
		return fmt.Sprintf("uploaded-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, uploads)
	require.Len(t, merged, 1)
	require.ElementsMatch(t, []string{"dup-a.ach", "dup-b.ach"}, merged[0].InputFilepaths)

	require.NotNil(t, uploadedFile)
	var entryCount int
	for _, b := range uploadedFile.Batches {
		entryCount += len(b.GetEntries())
	}
	// Intentional: moov-io/ach keeps each queued file's entries even when Nacha
	// content is identical across distinct file_ids. That moves both customers'
	// money and lets us emit FileUploaded for each file_id (not a silent dedupe).
	require.Equal(t, 2, entryCount, "duplicate-identity entries under distinct file_ids both upload")
}

func TestMerging_SameFileID_ConsumedTwice_UploadsOnce(t *testing.T) {
	// Re-queueing the same file_id overwrites storage; cutoff must treat it as
	// one input (one FileUploaded, one contribution to the merge).
	merging, _ := testMergingSetup(t, serviceShard("SD-same-id"))

	file := ach.NewFile()
	file.SetHeader(staticFileHeader())
	batch := ach.NewBatchPPD(mockBatchPPDHeader())
	entry := mockPPDEntryDetail()
	entry.Amount = 5000
	entry.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 11)
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	file.AddBatch(batch)
	require.NoError(t, file.Create())

	for i := 0; i < 2; i++ {
		err := merging.HandleXfer(context.Background(), incoming.ACHFile{
			FileID:   "same-id.ach",
			ShardKey: "SD-same-id",
			File:     file,
		})
		require.NoError(t, err)
	}

	var uploads int
	var uploadedFile *ach.File
	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, f *ach.File) (string, error) {
		uploads++
		uploadedFile = f
		return fmt.Sprintf("uploaded-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, uploads)
	require.Len(t, merged, 1)
	require.Equal(t, []string{"same-id.ach"}, merged[0].InputFilepaths)

	require.NotNil(t, uploadedFile)
	var entryCount int
	for _, b := range uploadedFile.Batches {
		entryCount += len(b.GetEntries())
	}
	require.Equal(t, 1, entryCount, "same file_id must only contribute one entry to the bank file")
}

func TestAccumulateMappings_DuplicateEntryIdentityQueuesBoth(t *testing.T) {
	dir := t.TempDir()
	m := &filesystemMerging{
		logger:  log.NewTestLogger(),
		storage: mustChest(t, dir),
	}

	file := ach.NewFile()
	file.SetHeader(staticFileHeader())
	batch := ach.NewBatchPPD(mockBatchPPDHeader())
	entry := mockPPDEntryDetail()
	entry.Amount = 99
	entry.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 7)
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	file.AddBatch(batch)
	require.NoError(t, file.Create())

	var buf bytes.Buffer
	require.NoError(t, ach.NewWriter(&buf).Write(file))
	require.NoError(t, m.storage.MkdirAll("iso"))
	require.NoError(t, m.storage.WriteFile("iso/first.ach", buf.Bytes()))
	require.NoError(t, m.storage.WriteFile("iso/second.ach", buf.Bytes()))

	mappings := make(entryFileMapping)
	seen := make(map[string]map[string]struct{})
	require.NoError(t, m.accumulateMappings(mappings, seen, "iso/first.ach"))
	require.NoError(t, m.accumulateMappings(mappings, seen, "iso/second.ach"))

	key := makeKey(batch.GetHeader(), entry)
	require.ElementsMatch(t, []string{"first.ach", "second.ach"}, mappings[key])
	require.Equal(t, []string{"first.ach", "second.ach"}, crossFileDuplicateIdentityFiles(mappings))
}

func TestAccumulateMappings_SameFileID_OncePerKey(t *testing.T) {
	// Same file_id must only occupy one slot per entry identity, even if the
	// path is accumulated twice (idempotent re-queue / re-read).
	dir := t.TempDir()
	m := &filesystemMerging{
		logger:  log.NewTestLogger(),
		storage: mustChest(t, dir),
	}

	file := ach.NewFile()
	file.SetHeader(staticFileHeader())
	batch := ach.NewBatchPPD(mockBatchPPDHeader())
	entry := mockPPDEntryDetail()
	entry.Amount = 40
	entry.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 3)
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	file.AddBatch(batch)
	require.NoError(t, file.Create())

	var buf bytes.Buffer
	require.NoError(t, ach.NewWriter(&buf).Write(file))
	require.NoError(t, m.storage.MkdirAll("iso"))
	require.NoError(t, m.storage.WriteFile("iso/only.ach", buf.Bytes()))

	mappings := make(entryFileMapping)
	seen := make(map[string]map[string]struct{})
	require.NoError(t, m.accumulateMappings(mappings, seen, "iso/only.ach"))
	require.NoError(t, m.accumulateMappings(mappings, seen, "iso/only.ach"))
	require.Len(t, mappings, 1)
	for _, names := range mappings {
		require.Equal(t, []string{"only.ach"}, names)
	}
	require.Empty(t, crossFileDuplicateIdentityFiles(mappings))
}

func TestFindInputFilepaths_DuplicateIdentityConsumesQueue(t *testing.T) {
	m := &filesystemMerging{logger: log.NewTestLogger()}

	// Build two merged batches that share entry identity (as merge does for dup traces).
	mkFile := func(batchNum int) *ach.File {
		f := ach.NewFile()
		f.SetHeader(staticFileHeader())
		bh := mockBatchPPDHeader()
		bh.BatchNumber = batchNum
		batch := ach.NewBatchPPD(bh)
		entry := mockPPDEntryDetail()
		entry.Amount = 50
		entry.SetTraceNumber(bh.ODFIIdentification, 9)
		batch.AddEntry(entry)
		require.NoError(t, batch.Create())
		f.AddBatch(batch)
		require.NoError(t, f.Create())
		return f
	}

	key := makeKey(mockBatchPPDHeader(), func() *ach.EntryDetail {
		e := mockPPDEntryDetail()
		e.Amount = 50
		e.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 9)
		return e
	}())
	mappings := entryFileMapping{
		key: {"dup-a.ach", "dup-b.ach"},
	}

	// One merged file containing both batches (two identical-identity entries).
	combined := mkFile(1)
	b2 := mkFile(2).Batches[0]
	combined.AddBatch(b2)
	require.NoError(t, combined.Create())

	out, leftover := m.findInputFilepaths(mappings, mergedFiles{
		{ACHFile: combined, UploadedFilename: "out.ach"},
	})
	require.Zero(t, leftover)
	require.ElementsMatch(t, []string{"dup-a.ach", "dup-b.ach"}, out[0].InputFilepaths)
}

func TestFindInputFilepaths_NilACHFile(t *testing.T) {
	m := &filesystemMerging{logger: log.NewTestLogger()}
	mappings := entryFileMapping{"key": {"input.ach"}}
	out, leftover := m.findInputFilepaths(mappings, mergedFiles{
		{ACHFile: nil, UploadedFilename: "x.ach"},
	})
	require.Len(t, out, 1)
	require.Empty(t, out[0].InputFilepaths)
	require.Equal(t, 1, leftover) // mapping unused
}

func TestMissingMappedInputFiles_EmptyNames(t *testing.T) {
	expected := []string{"", ".", string(filepath.Separator), "real.ach"}
	merged := mergedFiles{{InputFilepaths: []string{"real.ach"}}}
	require.Empty(t, missingMappedInputFiles(expected, merged))
}

func TestCountMappedInputFiles(t *testing.T) {
	require.Equal(t, 0, countMappedInputFiles(nil))
	require.Equal(t, 3, countMappedInputFiles(mergedFiles{
		{InputFilepaths: []string{"a", "b"}},
		{InputFilepaths: []string{"c"}},
	}))
}

func TestMerging_WithEachMerged_ManyDuplicateIdentitySample(t *testing.T) {
	// Many duplicate-identity inputs all upload and each file_id maps.
	merging, _ := testMergingSetup(t, serviceShard("SD-many-dup"))

	file := ach.NewFile()
	file.SetHeader(staticFileHeader())
	batch := ach.NewBatchPPD(mockBatchPPDHeader())
	entry := mockPPDEntryDetail()
	entry.Amount = 777
	entry.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 99)
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	file.AddBatch(batch)
	require.NoError(t, file.Create())

	const n = 25
	expected := make([]string, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("dup-%02d.ach", i)
		expected[i] = id
		err := merging.HandleXfer(context.Background(), incoming.ACHFile{
			FileID:   id,
			ShardKey: "SD-many-dup",
			File:     file,
		})
		require.NoError(t, err)
	}

	var uploads int
	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		uploads++
		return fmt.Sprintf("many-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, uploads)
	require.Len(t, merged, 1)
	require.ElementsMatch(t, expected, merged[0].InputFilepaths)
}

func TestMerging_DistinctEntries_StillUpload(t *testing.T) {
	// Unique entry identities must still merge and upload normally.
	merging, _ := testMergingSetup(t, serviceShard("SD-distinct"))
	enqueuePPD(t, merging, "SD-distinct", "a.ach", 1000, true, 1)
	enqueuePPD(t, merging, "SD-distinct", "b.ach", 2000, true, 2)

	var uploads int
	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		uploads++
		return fmt.Sprintf("out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, uploads)
	require.Len(t, merged, 1)
	require.ElementsMatch(t, []string{"a.ach", "b.ach"}, merged[0].InputFilepaths)
}

func serviceShard(name string) service.Shard {
	return service.Shard{Name: name}
}

type ccdOpts struct {
	fileID  string
	amount  int
	seq     int
	addenda string // empty = no addenda
	header  func(*ach.BatchHeader)
}

func enqueueCCD(t *testing.T, merging XferMerging, shardKey string, opts ccdOpts) {
	t.Helper()

	file := ach.NewFile()
	file.SetHeader(staticFileHeader())

	bh := mockBatchPPDHeader()
	bh.ServiceClassCode = ach.MixedDebitsAndCredits
	bh.StandardEntryClassCode = ach.CCD
	bh.CompanyEntryDescription = "PAYMENT"
	if opts.header != nil {
		opts.header(bh)
	}

	batch := ach.NewBatchCCD(bh)
	entry := mockPPDEntryDetail()
	entry.Amount = opts.amount
	entry.TransactionCode = ach.CheckingCredit
	entry.SetTraceNumber(bh.ODFIIdentification, opts.seq)
	if opts.addenda != "" {
		a := ach.NewAddenda05()
		a.PaymentRelatedInformation = opts.addenda
		entry.AddendaRecordIndicator = 1
		entry.AddAddenda05(a)
	}
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	file.AddBatch(batch)
	require.NoError(t, file.Create())

	err := merging.HandleXfer(context.Background(), incoming.ACHFile{
		FileID:   opts.fileID,
		ShardKey: shardKey,
		File:     file,
	})
	require.NoError(t, err)
}

func TestMerging_MergeEqualHeader_DifferentDiscretionaryData_MapsAll(t *testing.T) {
	// MergeDir combines batches with BatchHeader.Equal, which ignores
	// CompanyDiscretionaryData. Entries from the second file land under the
	// first file's header, so a mapping key built from header.String()[:87]
	// misses those inputs.
	const shard = "SD-disc"
	merging, _ := testMergingSetup(t, serviceShard(shard))

	enqueueCCD(t, merging, shard, ccdOpts{fileID: "disc-a.ach", amount: 1000, seq: 1})
	enqueueCCD(t, merging, shard, ccdOpts{
		fileID:  "disc-b.ach",
		amount:  2000,
		seq:     2,
		addenda: "INV*1001",
		header: func(bh *ach.BatchHeader) {
			bh.CompanyDiscretionaryData = "CODE1"
		},
	})

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("disc-out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Len(t, merged, 1)
	require.ElementsMatch(t, []string{"disc-a.ach", "disc-b.ach"}, merged[0].InputFilepaths)
}

func TestMerging_MergeEqualHeader_DifferentDescriptiveDate_MapsAll(t *testing.T) {
	const shard = "SD-descdate"
	merging, _ := testMergingSetup(t, serviceShard(shard))

	enqueueCCD(t, merging, shard, ccdOpts{fileID: "date-a.ach", amount: 1111, seq: 10})
	enqueueCCD(t, merging, shard, ccdOpts{
		fileID:  "date-b.ach",
		amount:  2222,
		seq:     11,
		addenda: "INV*1002",
		header: func(bh *ach.BatchHeader) {
			bh.CompanyDescriptiveDate = "Sep 01"
		},
	})

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("date-out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Len(t, merged, 1)
	require.ElementsMatch(t, []string{"date-a.ach", "date-b.ach"}, merged[0].InputFilepaths)
}

func TestMerging_MergeEqualHeader_CompanyNameCase_MapsAll(t *testing.T) {
	const shard = "SD-case"
	merging, _ := testMergingSetup(t, serviceShard(shard))

	enqueueCCD(t, merging, shard, ccdOpts{fileID: "case-a.ach", amount: 3333, seq: 20})
	enqueueCCD(t, merging, shard, ccdOpts{
		fileID:  "case-b.ach",
		amount:  4444,
		seq:     21,
		addenda: "INV*1003",
		header: func(bh *ach.BatchHeader) {
			bh.CompanyName = "acme corporation"
		},
	})

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("case-out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Len(t, merged, 1)
	require.ElementsMatch(t, []string{"case-a.ach", "case-b.ach"}, merged[0].InputFilepaths)
}

func TestMerging_MergeEqualHeader_DifferentOriginatorStatus_MapsAll(t *testing.T) {
	const shard = "SD-origstat"
	merging, _ := testMergingSetup(t, serviceShard(shard))

	enqueueCCD(t, merging, shard, ccdOpts{
		fileID: "orig-a.ach", amount: 5555, seq: 30,
		header: func(bh *ach.BatchHeader) {
			bh.OriginatorStatusCode = 1
		},
	})
	enqueueCCD(t, merging, shard, ccdOpts{
		fileID:  "orig-b.ach",
		amount:  6666,
		seq:     31,
		addenda: "INV*1004",
		header: func(bh *ach.BatchHeader) {
			bh.OriginatorStatusCode = 2
		},
	})

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("orig-out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Len(t, merged, 1)
	require.ElementsMatch(t, []string{"orig-a.ach", "orig-b.ach"}, merged[0].InputFilepaths)
}

func TestMerging_CCDAddenda_MixedWithNonAddenda_MapsAll(t *testing.T) {
	const shard = "SD-addenda-mix"
	merging, _ := testMergingSetup(t, serviceShard(shard))

	enqueueCCD(t, merging, shard, ccdOpts{fileID: "plain-1.ach", amount: 1000, seq: 1})
	enqueueCCD(t, merging, shard, ccdOpts{fileID: "addenda-a.ach", amount: 2000, seq: 2, addenda: "INV*2001"})
	enqueueCCD(t, merging, shard, ccdOpts{fileID: "plain-2.ach", amount: 3000, seq: 3})
	enqueueCCD(t, merging, shard, ccdOpts{fileID: "addenda-b.ach", amount: 4000, seq: 4, addenda: "INV*2002"})

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("mix-out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Len(t, merged, 1)
	require.ElementsMatch(t, []string{"plain-1.ach", "addenda-a.ach", "plain-2.ach", "addenda-b.ach"}, merged[0].InputFilepaths)
}

func TestMakeKey_MatchesMergeEqualNotFullHeaderString(t *testing.T) {
	entry := mockPPDEntryDetail()

	a := mockBatchPPDHeader()
	b := mockBatchPPDHeader()
	b.CompanyDiscretionaryData = "CODE1"
	b.CompanyDescriptiveDate = "Sep 01"
	b.SettlementDate = "123"
	b.OriginatorStatusCode = 2
	b.CompanyName = "acme corporation"
	b.BatchNumber = 99

	require.True(t, a.Equal(b), "precondition: merge would combine these batches")
	require.NotEqual(t, a.String()[:87], b.String()[:87], "precondition: full header strings differ")
	require.Equal(t, makeKey(a, entry), makeKey(b, entry), "mapping key must follow BatchHeader.Equal, not String()[:87]")
}
