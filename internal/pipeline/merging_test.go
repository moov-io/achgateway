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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestMerging__getCanceledFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "test-2024")
	require.NoError(t, os.MkdirAll(dir, 0777))

	name1 := fmt.Sprintf("%s.ach", base.ID())
	xfer1 := write(t, filepath.Join(dir, name1), nil)
	write(t, filepath.Join(dir, fmt.Sprintf("%s.canceled", xfer1)), nil)

	name2 := fmt.Sprintf("%s.ach", base.ID())
	write(t, filepath.Join(dir, name2), nil)

	name3 := fmt.Sprintf("%s.ach", base.ID())
	write(t, filepath.Join(dir, fmt.Sprintf("%s.canceled", name3)), nil)

	fs, err := storage.NewFilesystem(root)
	require.NoError(t, err)

	m := &filesystemMerging{
		storage: fs,
	}

	matches, err := m.getCanceledFiles(context.Background(), "test-2024")
	require.NoError(t, err)

	require.Len(t, matches, 2)
	require.ElementsMatch(t, []string{name1, name3}, matches)
}

func TestMerging__getNonCanceledMatches(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "test-2021")
	require.NoError(t, os.Mkdir(dir, 0777))

	xfer1 := write(t, filepath.Join(dir, fmt.Sprintf("%s.ach", base.ID())), nil)

	cancel1 := write(t, filepath.Join(dir, fmt.Sprintf("%s.ach.canceled", base.ID())), nil)

	xfer2 := write(t, filepath.Join(dir, fmt.Sprintf("%s.ach", base.ID())), nil)
	cancel2 := write(t, filepath.Join(dir, fmt.Sprintf("%s.canceled", xfer2)), nil)

	fs, err := storage.NewFilesystem(root)
	require.NoError(t, err)

	m := &filesystemMerging{
		storage: fs,
	}

	matches, err := m.getNonCanceledMatches(context.Background(), "test-2021")
	require.NoError(t, err)

	require.Len(t, matches, 1)
	require.Contains(t, matches[0], filepath.Join("test-2021", xfer1))

	require.NotContains(t, matches[0], filepath.Join("test-2021", cancel1))
	require.NotContains(t, matches[0], filepath.Join("test-2021", xfer2))
	require.NotContains(t, matches[0], filepath.Join("test-2021", cancel2))
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

func read(t testing.TB, where string) *ach.File {
	t.Helper()

	file, err := ach.ReadFile(where)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func write(t *testing.T, where string, contents []byte) string {
	t.Helper()
	err := os.WriteFile(where, contents, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, filename := filepath.Split(where)
	return filename
}

func TestMerging_fileAcceptor(t *testing.T) {
	name1 := fmt.Sprintf("%s.ach", base.ID())
	name2 := fmt.Sprintf("%s.ach", base.ID())
	json1 := fmt.Sprintf("%s.json", base.ID())

	output := fileAcceptor(nil)(name1)
	require.Equal(t, ach.AcceptFile, output)

	output = fileAcceptor([]string{name1})(name1)
	require.Equal(t, ach.SkipFile, output)

	output = fileAcceptor([]string{name1})(name2)
	require.Equal(t, ach.AcceptFile, output)

	output = fileAcceptor(nil)(json1)
	require.Equal(t, ach.SkipFile, output)

	output = fileAcceptor([]string{name1})(json1)
	require.Equal(t, ach.SkipFile, output)
}

func TestMerging_mappings(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "mergable"), 0777)
	copyFile(t, filepath.Join("testdata", "ppd-debit.ach"), filepath.Join(dir, "mergable", "ppd-debit.ach"))
	copyFile(t, filepath.Join("testdata", "ppd-debit.json"), filepath.Join(dir, "mergable", "ppd-debit.json"))
	copyFile(t, filepath.Join("testdata", "ppd-debit2.ach"), filepath.Join(dir, "mergable", "ppd-debit2.ach"))
	copyFile(t, filepath.Join("testdata", "ppd-debit3.ach"), filepath.Join(dir, "mergable", "ppd-debit3.ach"))
	copyFile(t, filepath.Join("testdata", "ppd-debit4.ach"), filepath.Join(dir, "mergable", "ppd-debit4.ach"))
	copyFile(t, filepath.Join("testdata", "duplicate-trace.ach"), filepath.Join(dir, "mergable", "duplicate-trace.ach"))

	// Canceled files
	err := os.WriteFile(filepath.Join(dir, "mergable", "foo2.ach"), nil, 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "mergable", "foo2.ach.canceled"), nil, 0600)
	require.NoError(t, err)

	fs, err := storage.NewFilesystem(dir)
	require.NoError(t, err)

	m := &filesystemMerging{
		logger: log.NewTestLogger(),
		cfg: service.UploadAgents{
			Agents: []service.UploadAgent{
				{
					ID:   "mock",
					Mock: &service.MockAgent{},
				},
			},
		},
		storage: fs,
		shard: service.Shard{
			UploadAgent: "mock",
		},
	}

	canceledFiles := []string{"foo2.ach"}
	mappings, err := m.buildDirMapping(".", canceledFiles)
	require.NoError(t, err)

	for it := mappings.Iterator(); it.Valid(); it.Next() {
		switch it.Value() {
		case "ppd-debit.ach", "duplicate-trace.ach":
			require.Contains(t, it.Key(), "076401255655291")
		case "ppd-debit2.ach":
			require.Contains(t, it.Key(), "076401255655292")
		case "ppd-debit3.ach":
			require.Contains(t, it.Key(), "076401255655293")
		case "ppd-debit4.ach":
			require.Contains(t, it.Key(), "076401255655294")
		}
	}

	ctx := context.Background()
	merged, err := m.WithEachMerged(ctx, func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("MAPPING-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Len(t, merged, 1)

	validateOpts := merged[0].ACHFile.GetValidation()
	require.NotNil(t, validateOpts)
	require.True(t, validateOpts.RequireABAOrigin)
	require.False(t, validateOpts.AllowZeroBatches)

	mapped := m.findInputFilepaths(mappings, merged)
	require.Len(t, mapped, 1)

	expected := []string{"duplicate-trace.ach", "ppd-debit.ach", "ppd-debit2.ach", "ppd-debit3.ach", "ppd-debit4.ach"}
	require.ElementsMatch(t, expected, mapped[0].InputFilepaths)
	require.Equal(t, "MAPPING-0.ach", mapped[0].UploadedFilename)
	require.Equal(t, 2, len(mapped[0].ACHFile.Batches))
}

func TestMerging__writeACHFile(t *testing.T) {
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

	file := read(t, filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	file.Header.ImmediateOrigin = "ABCDEFGHIJ"
	file.Header.ImmediateDestination = "123456780"

	fileID := base.ID()
	xfer := models.QueueACHFile{
		FileID:   fileID,
		ShardKey: "testing",
		File:     file,
	}
	xfer.SetValidation(&ach.ValidateOpts{
		BypassOriginValidation:      true,
		BypassDestinationValidation: true,
	})

	err = m.HandleXfer(context.Background(), incoming.ACHFile(xfer))
	require.NoError(t, err)

	// Verify the .ach and .json files were written
	mergableFilenames := getFilenames(t, m.storage, "mergable/testing")
	expected := []string{fmt.Sprintf("%s.ach", fileID), fmt.Sprintf("%s.json", fileID)}
	require.ElementsMatch(t, expected, mergableFilenames)

	var mergeConditions ach.Conditions
	opts := m.createMergeDirOptions(nil)
	opts.SubDirectories = true

	merged, err := ach.MergeDir("mergable", mergeConditions, opts)
	require.NoError(t, err)
	require.Len(t, merged, 1)

	validateOpts := merged[0].GetValidation()
	require.False(t, validateOpts.SkipAll)
	require.True(t, validateOpts.BypassOriginValidation)
	require.True(t, validateOpts.BypassDestinationValidation)

	var buf bytes.Buffer
	err = ach.NewWriter(&buf).Write(merged[0])
	require.NoError(t, err)

	// Verify the file pending contents
	require.True(t, bytes.HasPrefix(buf.Bytes(), []byte("101 123456780ABCDEFGHIJ")))
	require.Equal(t, "ABCDEFGHIJ", file.Header.ImmediateOrigin)
	require.Equal(t, "123456780", file.Header.ImmediateDestination)

	buf.Reset() // zero out
	err = ach.NewWriter(&buf).Write(merged[0])
	require.NoError(t, err)

	require.True(t, bytes.HasPrefix(buf.Bytes(), []byte("101 123456780ABCDEFGHIJ")))
	require.Equal(t, "ABCDEFGHIJ", merged[0].Header.ImmediateOrigin)
	require.Equal(t, "123456780", merged[0].Header.ImmediateDestination)
}

func getFilenames(t *testing.T, fsys fs.FS, dir string) []string {
	t.Helper()

	f, ok := fsys.(fs.ReadDirFS)
	if !ok {
		t.Fatalf("unexpected %T wanted fs.ReadDirFS", fsys)
	}

	items, err := f.ReadDir(dir)
	require.NoError(t, err)

	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].Name()
	}
	return out
}

func TestMerging__saveMergedFile(t *testing.T) {
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

	file := read(t, filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	file.SetValidation(&ach.ValidateOpts{
		RequireABAOrigin: true,
	})

	require.NoError(t, m.storage.MkdirAll("uploaded"))
	err = m.saveMergedFile(context.Background(), "uploaded", file)
	require.NoError(t, err)

	mergableFilenames := getFilenames(t, m.storage, "uploaded")
	expected := []string{
		"bb844ebc5b7f53a447a8bcff0c5a116b92b978657fddcf0d4f54b7ed991fa8b7.ach", // sha256 hash
		"bb844ebc5b7f53a447a8bcff0c5a116b92b978657fddcf0d4f54b7ed991fa8b7.json",
	}
	require.ElementsMatch(t, expected, mergableFilenames)
}
