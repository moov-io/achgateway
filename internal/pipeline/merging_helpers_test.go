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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func mustChest(t *testing.T, dir string) storage.Chest {
	t.Helper()
	chest, err := storage.New(storage.Config{
		Filesystem: storage.FilesystemConfig{Directory: dir},
	})
	require.NoError(t, err)
	return chest
}

func testMergingSetup(t *testing.T, shard service.Shard) (XferMerging, string) {
	t.Helper()
	dir := t.TempDir()
	if shard.Name == "" {
		shard.Name = "SD-test"
	}
	if shard.UploadAgent == "" {
		shard.UploadAgent = "mock"
	}
	cfg := service.UploadAgents{
		Agents: []service.UploadAgent{
			{ID: "mock", Mock: &service.MockAgent{}},
		},
		Merging: service.Merging{
			Storage: storage.Config{
				Filesystem: storage.FilesystemConfig{Directory: dir},
			},
		},
	}
	merging, err := NewMerging(log.NewTestLogger(), shard, cfg)
	require.NoError(t, err)
	return merging, dir
}

func enqueuePPD(t *testing.T, merging XferMerging, shardKey, fileID string, amount int, credit bool, seq int) {
	t.Helper()
	file := ach.NewFile()
	file.SetHeader(staticFileHeader())
	bh := mockBatchPPDHeader()
	if credit {
		bh.ServiceClassCode = ach.CreditsOnly
	} else {
		bh.ServiceClassCode = ach.DebitsOnly
	}
	batch := ach.NewBatchPPD(bh)
	entry := mockPPDEntryDetail()
	entry.Amount = amount
	if credit {
		entry.TransactionCode = ach.CheckingCredit
	} else {
		entry.TransactionCode = ach.CheckingDebit
	}
	entry.SetTraceNumber(bh.ODFIIdentification, seq)
	batch.AddEntry(entry)
	require.NoError(t, batch.Create())
	file.AddBatch(batch)
	require.NoError(t, file.Create())

	err := merging.HandleXfer(context.Background(), incoming.ACHFile{
		FileID:   fileID,
		ShardKey: shardKey,
		File:     file,
	})
	require.NoError(t, err)
}

func findIsolatedShardDirs(t *testing.T, root, shard string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	prefix := shard + "-"
	var out []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			out = append(out, e.Name())
		}
	}
	return out
}

func mockFileHeader() ach.FileHeader {
	fh := ach.NewFileHeader()
	fh.ImmediateDestination = "231380104"
	fh.ImmediateOrigin = "121042882"
	fh.FileCreationDate = time.Now().AddDate(0, 0, 1).Format("060102") // YYMMDD
	fh.ImmediateDestinationName = "Federal Reserve Bank"
	fh.ImmediateOriginName = "My Bank Name"
	return fh
}

func staticFileHeader() ach.FileHeader {
	fh := mockFileHeader()
	fh.FileCreationDate = "230421" // YYMMDD
	fh.FileCreationTime = "1201"   // HHMM
	return fh
}

func mockBatchPPDHeader() *ach.BatchHeader {
	bh := ach.NewBatchHeader()
	bh.ServiceClassCode = ach.CreditsOnly
	bh.StandardEntryClassCode = ach.PPD
	bh.CompanyName = "ACME Corporation"
	bh.CompanyIdentification = "121042882"
	bh.CompanyEntryDescription = "PAYROLL"
	bh.EffectiveEntryDate = time.Now().AddDate(0, 0, 1).Format("060102") // YYMMDD
	bh.ODFIIdentification = "12104288"
	return bh
}

func mockPPDEntryDetail() *ach.EntryDetail {
	entry := ach.NewEntryDetail()
	entry.TransactionCode = ach.CheckingCredit
	entry.SetRDFI("231380104")
	entry.DFIAccountNumber = "123456789"
	entry.Amount = 100000000
	entry.IndividualName = "Wade Arnold"
	entry.SetTraceNumber(mockBatchPPDHeader().ODFIIdentification, 1)
	entry.Category = ach.CategoryForward
	return entry
}

func TestFileAcceptor_CanceledSuffixPath(t *testing.T) {
	accept := fileAcceptor([]string{"a.ach"})
	require.Equal(t, ach.SkipFile, accept("mergable/a.ach.canceled"))
	require.Equal(t, ach.SkipFile, accept("a.ach"))
	require.Equal(t, ach.AcceptFile, accept("other.ach"))
	require.Equal(t, ach.SkipFile, accept("other.txt"))
	require.Equal(t, ach.AcceptFile, accept("path/to/file.ACH")) // case-insensitive .ach
}

func TestStripACHExtension(t *testing.T) {
	stem, ok := stripACHExtension("foo.ach")
	require.True(t, ok)
	require.Equal(t, "foo", stem)

	stem, ok = stripACHExtension("foo.ACH")
	require.True(t, ok)
	require.Equal(t, "foo", stem)

	_, ok = stripACHExtension("foo.json")
	require.False(t, ok)

	require.Equal(t, "id", trimACHExtension("id.ach"))
	require.Equal(t, "id", trimACHExtension("id.ACH"))
	require.Equal(t, "id.txt", trimACHExtension("id.txt"))
}

func TestGetNonCanceledMatches_StemWithoutAchSuffix(t *testing.T) {
	// Canceled as "name.canceled" (stem without .ach) must still exclude "name.ach".
	root := t.TempDir()
	dir := filepath.Join(root, "stem")
	require.NoError(t, os.MkdirAll(dir, 0777))

	live := write(t, filepath.Join(dir, "live.ach"), []byte("x"))
	canceledBase := "gone"
	write(t, filepath.Join(dir, canceledBase+".ach"), []byte("x"))
	write(t, filepath.Join(dir, canceledBase+".canceled"), []byte("x"))

	fs, err := storage.NewFilesystem(root)
	require.NoError(t, err)
	m := &filesystemMerging{storage: fs}

	matches, err := m.getNonCanceledMatches(context.Background(), "stem")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Contains(t, matches[0], live)
	require.NotContains(t, matches[0], canceledBase)
}

func TestGetNonCanceledMatches_ManyCanceled(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "bulk")
	require.NoError(t, os.MkdirAll(dir, 0777))

	// 50 live files and 50 canceled
	for i := 0; i < 50; i++ {
		write(t, filepath.Join(dir, fmt.Sprintf("live-%d.ach", i)), []byte("x"))
		write(t, filepath.Join(dir, fmt.Sprintf("dead-%d.ach", i)), []byte("x"))
		write(t, filepath.Join(dir, fmt.Sprintf("dead-%d.ach.canceled", i)), []byte("x"))
	}

	fs, err := storage.NewFilesystem(root)
	require.NoError(t, err)
	m := &filesystemMerging{storage: fs}

	matches, err := m.getNonCanceledMatches(context.Background(), "bulk")
	require.NoError(t, err)
	require.Len(t, matches, 50)
	for _, p := range matches {
		require.Contains(t, p, "live-")
		require.NotContains(t, p, "dead-")
	}
}
