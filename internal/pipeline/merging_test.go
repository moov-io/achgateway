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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestMerging__getNonCanceledMatches(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "test-2021"), 0777))

	write := func(filename string) string {
		err := os.WriteFile(filepath.Join(dir, "test-2021", filename), nil, 0600)
		if err != nil {
			t.Fatal(err)
		}
		return filename
	}

	xfer1 := write(fmt.Sprintf("%s.ach", base.ID()))

	cancel1 := write(fmt.Sprintf("%s.ach.canceled", base.ID()))

	xfer2 := write(fmt.Sprintf("%s.ach", base.ID()))
	cancel2 := write(fmt.Sprintf("%s.canceled", xfer2))

	fs, err := storage.NewFilesystem(dir)
	require.NoError(t, err)

	m := &filesystemMerging{
		storage: fs,
	}

	matches, err := m.getNonCanceledMatches("test-2021")
	require.NoError(t, err)

	require.Len(t, matches, 1)
	require.Contains(t, matches[0], filepath.Join("test-2021", xfer1))

	require.NotContains(t, matches[0], filepath.Join("test-2021", cancel1))
	require.NotContains(t, matches[0], filepath.Join("test-2021", xfer2))
	require.NotContains(t, matches[0], filepath.Join("test-2021", cancel2))
}

func TestMerging_makeIndices(t *testing.T) {
	indices := makeIndices(122, 5)
	expected := []int{0, 24, 48, 72, 96, 120, 122}
	require.Equal(t, expected, indices)

	indices = makeIndices(500, 1)
	expected = []int{500}
	require.Equal(t, expected, indices)
}

func copyFile(t *testing.T, src, dest string) {
	t.Helper()

	s, err := os.Open(src)
	require.NoError(t, err)
	defer s.Close()

	d, err := os.Create(dest)
	require.NoError(t, err)
	defer d.Close()

	n, err := io.Copy(d, s)
	require.NoError(t, err)
	require.Greater(t, n, int64(0))
}

func TestMerging_chunkFilesTogether(t *testing.T) {
	dir := t.TempDir()

	copyFile(t, filepath.Join("..", "..", "testdata", "ppd-debit.ach"), filepath.Join(dir, "ppd-debit.ach"))
	copyFile(t, filepath.Join("..", "..", "testdata", "ppd-debit2.ach"), filepath.Join(dir, "ppd-debit2.ach"))

	fs, err := storage.NewFilesystem(dir)
	require.NoError(t, err)

	m := &filesystemMerging{
		logger:  log.NewTestLogger(),
		storage: fs,
	}

	indices := makeIndices(2, 1)
	matches := []string{"ppd-debit.ach", "ppd-debit2.ach"}
	var conditions ach.Conditions
	merged, err := m.chunkFilesTogether(indices, matches, conditions)
	require.NoError(t, err)
	require.Len(t, merged, 1)
	require.Len(t, merged[0].Batches[0].GetEntries(), 2)
}

func TestMerging__writeACHFile(t *testing.T) {
	dir := t.TempDir()
	fs, err := storage.NewFilesystem(dir)
	require.NoError(t, err)

	m := &filesystemMerging{
		logger: log.NewTestLogger(),
		shard: service.Shard{
			Name: "testing",
		},
		storage: fs,
	}

	file, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)

	file.Header.ImmediateOrigin = "ABCDEFGHIJ"
	file.Header.ImmediateDestination = "123456780"

	xfer := models.QueueACHFile{
		FileID:   base.ID(),
		ShardKey: "testing",
		File:     file,
	}
	xfer.SetValidation(&ach.ValidateOpts{
		BypassOriginValidation:      true,
		BypassDestinationValidation: true,
	})

	err = m.HandleXfer(incoming.ACHFile(xfer))
	require.NoError(t, err)

	// Read the pending file
	pendingFile, err := m.readFile(filepath.Join("mergable", "testing", fmt.Sprintf("%s.ach", xfer.FileID)))
	require.NoError(t, err)
	require.NotNil(t, pendingFile.GetValidation())

	var buf bytes.Buffer
	err = ach.NewWriter(&buf).Write(pendingFile)
	require.NoError(t, err)

	// Verify the file pending contents
	require.True(t, bytes.HasPrefix(buf.Bytes(), []byte("101 123456780ABCDEFGHIJ")))
	require.Equal(t, "ABCDEFGHIJ", pendingFile.Header.ImmediateOrigin)
	require.Equal(t, "123456780", pendingFile.Header.ImmediateDestination)

	merged, err := ach.MergeFiles([]*ach.File{pendingFile})
	require.NoError(t, err)
	require.Len(t, merged, 1)
	require.NotNil(t, merged[0].GetValidation())

	buf.Reset() // zero out
	err = ach.NewWriter(&buf).Write(merged[0])
	require.NoError(t, err)

	require.True(t, bytes.HasPrefix(buf.Bytes(), []byte("101 123456780ABCDEFGHIJ")))
	require.Equal(t, "ABCDEFGHIJ", merged[0].Header.ImmediateOrigin)
	require.Equal(t, "123456780", merged[0].Header.ImmediateDestination)
}
