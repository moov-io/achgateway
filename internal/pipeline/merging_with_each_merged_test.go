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
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestMerging_WithEachMerged_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	logger := log.NewTestLogger()
	shard := service.Shard{
		Name:        "SD-empty",
		UploadAgent: "mock",
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

	merging, err := NewMerging(logger, shard, cfg)
	require.NoError(t, err)

	// Ensure empty mergable directory exists
	m := merging.(*filesystemMerging)
	require.NoError(t, m.storage.MkdirAll("mergable/SD-empty"))

	var callbackRan bool
	merged, err := merging.WithEachMerged(context.Background(), func(context.Context, int, upload.Agent, *ach.File) (string, error) {
		callbackRan = true
		return "", nil
	})
	require.NoError(t, err)
	require.Empty(t, merged)
	require.False(t, callbackRan, "callback should not run for empty merge")
	// Successful empty merge may still isolate then remove the directory.
	require.Empty(t, findIsolatedShardDirs(t, dir, "SD-empty"))
}

func TestMerging_MaxLinesCondition(t *testing.T) {
	merging, _ := testMergingSetup(t, service.Shard{
		Name: "SD-lines",
		Mergable: service.MergableConfig{
			Conditions: &ach.Conditions{
				// FileHeader+FileControl+BatchHeader+BatchControl = 4, plus 1 entry = 5.
				// A second entry would be 6 lines, so MaxLines=5 forces one entry per file.
				MaxLines: 5,
			},
		},
	})

	// Enqueue several single-entry files with distinct traces
	for i := 0; i < 5; i++ {
		enqueuePPD(t, merging, "SD-lines", fmt.Sprintf("line-%d.ach", i), 100*(i+1), true, i+1)
	}

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Greater(t, len(merged), 1, "MaxLines should force multiple output files")

	var totalEntries int
	for _, mf := range merged {
		for _, b := range mf.ACHFile.Batches {
			totalEntries += len(b.GetEntries())
		}
	}
	require.Equal(t, 5, totalEntries)
}

func TestMerging_MaxLinesCondition_AllInputsMapped(t *testing.T) {
	merging, _ := testMergingSetup(t, service.Shard{
		Name: "SD-lines-map",
		Mergable: service.MergableConfig{
			Conditions: &ach.Conditions{
				MaxLines: 5,
			},
		},
	})

	var expected []string
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("line-%d.ach", i)
		expected = append(expected, id)
		enqueuePPD(t, merging, "SD-lines-map", id, 100*(i+1), true, i+1)
	}

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Greater(t, len(merged), 1)

	var got []string
	for _, mf := range merged {
		got = append(got, mf.InputFilepaths...)
	}
	require.ElementsMatch(t, expected, got)
	require.Empty(t, missingMappedInputFiles(expected, merged))
}

func TestMerging_WithEachMerged_FlattenBatches(t *testing.T) {
	// Two separate batches should flatten into one when FlattenBatches is enabled.
	merging, _ := testMergingSetup(t, service.Shard{
		Name: "SD-flat",
		Mergable: service.MergableConfig{
			FlattenBatches: &service.FlattenBatches{},
		},
	})

	// Two files with same company header fields so FlattenBatches can combine
	for i := 0; i < 2; i++ {
		enqueuePPD(t, merging, "SD-flat", fmt.Sprintf("flat-%d.ach", i), 1000*(i+1), true, i+1)
	}

	const inputBatches = 2
	var outputBatchCount int
	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, f *ach.File) (string, error) {
		outputBatchCount += len(f.Batches)
		return fmt.Sprintf("flat-out-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, merged)
	// Two single-batch inputs with matching company headers should flatten into
	// fewer batches than the input count (typically one combined batch).
	require.Less(t, outputBatchCount, inputBatches, "FlattenBatches should combine input batches")
	// All inputs mapped
	var got []string
	for _, mf := range merged {
		got = append(got, mf.InputFilepaths...)
	}
	require.ElementsMatch(t, []string{"flat-0.ach", "flat-1.ach"}, got)
}

func TestMerging_WithEachMerged_UploadErrorContinues(t *testing.T) {
	// One upload failure must not prevent remaining files from uploading.
	merging, _ := testMergingSetup(t, service.Shard{
		Name: "SD-uperr",
		Mergable: service.MergableConfig{
			Conditions: &ach.Conditions{MaxLines: 5}, // force multiple output files
		},
	})

	const enqueued = 3
	for i := 0; i < enqueued; i++ {
		enqueuePPD(t, merging, "SD-uperr", fmt.Sprintf("up-%d.ach", i), 100*(i+1), true, i+1)
	}

	// WithEachMerged invokes the upload callback sequentially (one file at a time).
	var attempts []int
	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		attempts = append(attempts, idx)
		if idx == 0 {
			return "", fmt.Errorf("simulated upload failure")
		}
		return fmt.Sprintf("ok-%d.ach", idx), nil
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "simulated upload failure")
	require.Greater(t, len(attempts), 1, "later files must still be attempted after first upload error")
	require.Contains(t, attempts, 0, "failed idx must have been attempted")
	require.Contains(t, attempts, 1, "idx 1 must have been attempted after idx 0 failed")
	require.NotEmpty(t, merged, "successful uploads still returned")
	for _, mf := range merged {
		require.NotEqual(t, "", mf.UploadedFilename)
		require.NotContains(t, mf.UploadedFilename, "simulated")
		require.NotEqual(t, "ok-0.ach", mf.UploadedFilename)
	}
}

func TestMerging_WithEachMerged_MaxDollarAmountPerSide(t *testing.T) {
	// With ach v1.63.1+, MaxDollarAmount is per debit/credit side. $80 debit + $80 credit
	// under a $100 cap must stay in one merged file (not split by combined total).
	merging, _ := testMergingSetup(t, service.Shard{
		Name: "SD-dollar",
		Mergable: service.MergableConfig{
			Conditions: &ach.Conditions{
				MaxDollarAmount: 100_00,
			},
		},
	})

	enqueuePPD(t, merging, "SD-dollar", "debit.ach", 80_00, false, 1)
	enqueuePPD(t, merging, "SD-dollar", "credit.ach", 80_00, true, 2)

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, f *ach.File) (string, error) {
		return fmt.Sprintf("dollar-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.Len(t, merged, 1, "debit and credit sides limited independently")
	require.Equal(t, 80_00, merged[0].ACHFile.Control.TotalDebitEntryDollarAmountInFile)
	require.Equal(t, 80_00, merged[0].ACHFile.Control.TotalCreditEntryDollarAmountInFile)
	require.ElementsMatch(t, []string{"debit.ach", "credit.ach"}, merged[0].InputFilepaths)
}

func TestMerging_WithEachMerged_MergeDirError(t *testing.T) {
	// Invalid ACH fails ach.MergeDir; WithEachMerged must surface the error
	// without hanging, without uploading, and without destroying the isolated
	// input file(s) — operators need them for recovery.
	merging, root := testMergingSetup(t, service.Shard{Name: "SD-adv"})
	m := merging.(*filesystemMerging)

	require.NoError(t, m.storage.MkdirAll("mergable/SD-adv"))
	require.NoError(t, m.storage.WriteFile("mergable/SD-adv/bad.ach", []byte("not-valid-ach\n")))

	var uploads int
	merged, err := merging.WithEachMerged(context.Background(), func(context.Context, int, upload.Agent, *ach.File) (string, error) {
		uploads++
		return "x.ach", nil
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to merge files")
	require.Zero(t, uploads)
	require.Empty(t, merged)

	// Isolated cutoff directory must still contain the input ACH.
	isolated := findIsolatedShardDirs(t, root, "SD-adv")
	require.Len(t, isolated, 1)
	require.FileExists(t, filepath.Join(root, isolated[0], "bad.ach"))
}

func TestMerging_WithEachMerged_MergeDirErrorPreservesGoodInputs(t *testing.T) {
	// One poison file must not cause destruction of valid siblings in the same
	// isolated cutoff directory when MergeDir fails closed (nil files + error).
	merging, root := testMergingSetup(t, service.Shard{Name: "SD-poison"})
	m := merging.(*filesystemMerging)

	enqueuePPD(t, merging, "SD-poison", "good.ach", 2500, true, 1)
	require.NoError(t, m.storage.WriteFile("mergable/SD-poison/poison.ach", []byte("not-valid-ach\n")))

	var uploads int
	merged, err := merging.WithEachMerged(context.Background(), func(context.Context, int, upload.Agent, *ach.File) (string, error) {
		uploads++
		return "x.ach", nil
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "unable to merge files")
	require.Zero(t, uploads)
	require.Empty(t, merged)

	isolated := findIsolatedShardDirs(t, root, "SD-poison")
	require.Len(t, isolated, 1)
	isoPath := filepath.Join(root, isolated[0])
	require.FileExists(t, filepath.Join(isoPath, "good.ach"), "valid input must survive merge failure")
	require.FileExists(t, filepath.Join(isoPath, "poison.ach"), "failed input must survive for forensics")
}

func TestMerging_WithEachMerged_MissingUploadAgent(t *testing.T) {
	dir := t.TempDir()
	shard := service.Shard{
		Name:        "SD-noagent",
		UploadAgent: "does-not-exist",
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

	enqueuePPD(t, merging, "SD-noagent", "a.ach", 1000, true, 1)

	_, err = merging.WithEachMerged(context.Background(), func(context.Context, int, upload.Agent, *ach.File) (string, error) {
		return "x.ach", nil
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "merging agent")
}

func TestMerging_WithEachMerged_BadValidateOptsSidecar(t *testing.T) {
	// Corrupt ValidateOpts JSON next to a valid ACH should surface as a mapping
	// error but still allow upload of the merged file.
	merging, _ := testMergingSetup(t, service.Shard{Name: "SD-badjson"})
	enqueuePPD(t, merging, "SD-badjson", "good.ach", 1500, true, 1)

	m := merging.(*filesystemMerging)
	// Overwrite any sidecar (or create one) with invalid JSON
	require.NoError(t, m.storage.WriteFile("mergable/SD-badjson/good.json", []byte("{not-json")))

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("out-%d.ach", idx), nil
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "validate opts")
	require.NotEmpty(t, merged, "merged file still uploaded despite mapping error")
}

func TestMerging_WithEachMerged_ManyOutputFiles(t *testing.T) {
	// MaxLines forces one entry per output file; ensure a larger cutoff still
	// maps and uploads every input.
	merging, _ := testMergingSetup(t, service.Shard{
		Name: "SD-many",
		Mergable: service.MergableConfig{
			Conditions: &ach.Conditions{MaxLines: 5},
		},
	})

	const n = 12
	for i := 0; i < n; i++ {
		enqueuePPD(t, merging, "SD-many", fmt.Sprintf("p-%02d.ach", i), 100*(i+1), true, i+1)
	}

	merged, err := merging.WithEachMerged(context.Background(), func(_ context.Context, idx int, _ upload.Agent, _ *ach.File) (string, error) {
		return fmt.Sprintf("prog-%d.ach", idx), nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, merged)
	var got []string
	expected := make([]string, 0, n)
	for i := 0; i < n; i++ {
		expected = append(expected, fmt.Sprintf("p-%02d.ach", i))
	}
	for _, mf := range merged {
		got = append(got, mf.InputFilepaths...)
	}
	require.ElementsMatch(t, expected, got, "every input must map regardless of output-file split")
}
