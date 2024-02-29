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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/strx"
	"github.com/moov-io/base/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"
)

// XferMerging represents logic for accepting ACH files to be merged together.
//
// The idea is to take Xfers and store them on a filesystem (or other durable storage)
// prior to a cutoff window. The specific storage could be based on the FileHeader.
//
// On the cutoff trigger WithEachMerged is called to merge files together and offer
// each merged file for an upload.
type XferMerging interface {
	HandleXfer(ctx context.Context, xfer incoming.ACHFile) error
	HandleCancel(ctx context.Context, cancel incoming.CancelACHFile) error

	WithEachMerged(ctx context.Context, f func(context.Context, int, upload.Agent, *ach.File) (string, error)) (*processedFiles, error)
}

func NewMerging(logger log.Logger, shard service.Shard, cfg service.UploadAgents) (XferMerging, error) {
	dir := strx.Or(
		cfg.Merging.Storage.Filesystem.Directory,
		cfg.Merging.Directory,
		"storage", // default directory
	)
	cfg.Merging.Storage.Filesystem.Directory = dir

	storage, err := storage.New(cfg.Merging.Storage)
	if err != nil {
		return nil, fmt.Errorf("problem creating %s: %w", dir, err)
	}

	return &filesystemMerging{
		logger:  logger,
		cfg:     cfg,
		storage: storage,
		shard:   shard,
	}, nil
}

type filesystemMerging struct {
	logger  log.Logger
	cfg     service.UploadAgents
	storage storage.Chest
	shard   service.Shard
}

func (m *filesystemMerging) HandleXfer(ctx context.Context, xfer incoming.ACHFile) error {
	if err := m.writeACHFile(ctx, xfer); err != nil {
		telemetry.RecordError(ctx, err)

		return m.logger.LogErrorf("problem writing ACH file: %v", err).Err()
	}
	return nil
}

func (m *filesystemMerging) writeACHFile(ctx context.Context, xfer incoming.ACHFile) error {
	// First, write the Nacha formatted file to disk
	var buf bytes.Buffer
	if err := ach.NewWriter(&buf).Write(xfer.File); err != nil {
		return err
	}
	path := filepath.Join("mergable", m.shard.Name, fmt.Sprintf("%s.ach", xfer.FileID))
	if err := m.storage.WriteFile(path, buf.Bytes()); err != nil {
		return err
	}

	// Second, write ValidateOpts to disk as well
	if opts := xfer.File.GetValidation(); opts != nil {
		buf.Reset()
		if err := json.NewEncoder(&buf).Encode(opts); err != nil {
			m.logger.Warn().With(log.Fields{
				"fileID":   log.String(xfer.FileID),
				"shardKey": log.String(xfer.ShardKey),
			}).Logf("ERROR encoding ValidateOpts: %v", err)
		}
		path := filepath.Join("mergable", m.shard.Name, fmt.Sprintf("%s.json", xfer.FileID))
		if err := m.storage.WriteFile(path, buf.Bytes()); err != nil {
			m.logger.Warn().With(log.Fields{
				"fileID":   log.String(xfer.FileID),
				"shardKey": log.String(xfer.ShardKey),
			}).Logf("ERROR writing ValidateOpts: %v", err)
		}
	}

	return nil
}

func (m *filesystemMerging) HandleCancel(ctx context.Context, cancel incoming.CancelACHFile) error {
	path := filepath.Join("mergable", m.shard.Name, fmt.Sprintf("%s.ach", cancel.FileID))

	// Write the canceled File
	err := m.storage.ReplaceFile(path, path+".canceled")
	if err != nil {
		telemetry.RecordError(ctx, err)
	}
	return err
}

func (m *filesystemMerging) isolateMergableDir(ctx context.Context) (string, error) {
	newdir := filepath.Join(fmt.Sprintf("%s-%v", m.shard.Name, time.Now().Format("20060102-150405")))

	_, span := telemetry.StartSpan(ctx, "isolate-mergable-dir", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.String("achgateway.dir", newdir),
	))
	defer span.End()

	// Otherwise attempt to isolate the directory
	return newdir, m.storage.ReplaceDir(filepath.Join("mergable", m.shard.Name), newdir)
}

func (m *filesystemMerging) getNonCanceledMatches(ctx context.Context, dir string) ([]string, error) {
	_, span := telemetry.StartSpan(ctx, "get-non-canceled-matches", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.String("achgateway.dir", dir),
	))
	defer span.End()

	positiveMatches, err := m.storage.Glob(dir + "/*.ach")
	if err != nil {
		return nil, err
	}
	negativeMatches, err := m.storage.Glob(dir + "/*.canceled")
	if err != nil {
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("achgateway.positive_matches", len(positiveMatches)),
		attribute.Int("achgateway.negative_matches", len(negativeMatches)),
	)

	var out []string
	for i := range positiveMatches {
		exclude := false
		for j := range negativeMatches {
			// We match when a "XXX.ach.canceled" filepath exists and so we can't
			// include "XXX.ach" has a filepath from this function.
			if strings.HasPrefix(negativeMatches[j].RelativePath, positiveMatches[i].RelativePath) {
				exclude = true
				break
			}
		}
		if !exclude {
			out = append(out, positiveMatches[i].RelativePath)
		}
	}
	return out, nil
}

type processedFiles struct {
	shardKey string
	files    mergedFiles
}

func newProcessedFiles(shardKey string, merged mergedFiles) *processedFiles {
	out := &processedFiles{
		shardKey: shardKey,
		files:    merged,
	}
	for i := range out.files {
		for j := range out.files[i].Names {
			// each match follows $path/$fileID.ach
			out.files[i].Names[j] = strings.TrimSuffix(filepath.Base(out.files[i].Names[j]), ".ach")
		}
	}
	return out
}

func (m *filesystemMerging) readFile(path string) (*ach.File, error) {
	file, err := m.storage.Open(path)
	if err != nil {
		return nil, err
	}
	if file != nil {
		defer file.Close()
	}

	r := ach.NewReader(file)

	// Attempt to read ValidateOpts
	optsFile, _ := m.storage.Open(strings.Replace(path, ".ach", ".json", -1))
	if optsFile != nil {
		defer optsFile.Close()

		var opts ach.ValidateOpts
		json.NewDecoder(optsFile).Decode(&opts)

		r.SetValidation(&opts)
	}

	// Parse the Nacha formatted file
	f, err := r.Read()
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// namedFiles is a set of two arrays containing filenames and ACH files.
//
// Both arrays are the same length and each index corresponds to the other array.
type namedFiles struct {
	Names    []string
	ACHFiles []*ach.File
}

func (m *filesystemMerging) readFiles(paths []string) (namedFiles, error) {
	out := namedFiles{
		Names:    make([]string, len(paths)),
		ACHFiles: make([]*ach.File, len(paths)),
	}
	for i := range paths {
		file, err := m.readFile(paths[i])
		if err != nil {
			return namedFiles{}, fmt.Errorf("reading %s failed: %w", paths[i], err)
		}
		_, out.Names[i] = filepath.Split(paths[i])
		out.ACHFiles[i] = file
	}
	return out, nil
}

func (m *filesystemMerging) WithEachMerged(ctx context.Context, f func(context.Context, int, upload.Agent, *ach.File) (string, error)) (*processedFiles, error) {
	processed := &processedFiles{}

	// move the current directory so it's isolated and easier to debug later on
	dir, err := m.isolateMergableDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("problem isolating newdir=%s error=%v", dir, err)
	}

	_, span := telemetry.StartSpan(ctx, "with-each-merged", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.String("achgateway.dir", dir),
	))
	defer span.End()

	matches, err := m.getNonCanceledMatches(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("problem with %s glob: %v", dir, err)
	}

	logger := m.logger.Set("shardName", log.String(m.shard.Name))

	dirNames := strings.Join(directoryNames(matches), ", ")
	logger.Logf("found %d matching ACH files in %v", len(matches), dirNames)

	var el base.ErrorList

	// Merge files together in groups
	var mergeConditions ach.Conditions
	if m.shard.Mergable.Conditions != nil {
		mergeConditions = *m.shard.Mergable.Conditions
	}

	groupSize := 100
	if m.shard.Mergable.MergeInGroupsOf > 0 {
		groupSize = m.shard.Mergable.MergeInGroupsOf
	}
	indices := makeIndices(len(matches), len(matches)/groupSize)
	files, err := m.chunkFilesTogether(ctx, indices, matches, mergeConditions)
	if err != nil {
		el.Add(fmt.Errorf("unable to merge files: %v", err))
	}

	if len(matches) > 0 {
		logger.Logf("merged %d files into %d files", len(matches), len(files))
	}

	// Remove the directory if there are no files, otherwise setup an inner dir for the uploaded file.
	if len(files) == 0 {
		// delete the new directory as there's nothing to merge
		if err := m.storage.RmdirAll(dir); err != nil {
			el.Add(err)
		}
	} else {
		dir = filepath.Join(dir, "uploaded")
		m.storage.MkdirAll(dir)
	}

	// Grab our upload Agent
	agent, err := upload.New(m.logger, m.cfg, m.shard.UploadAgent)
	if err != nil {
		return processed, fmt.Errorf("%s merging agent: %v", m.shard.Name, err)
	}
	logger.Logf("found %T agent", agent)

	// Write each file to our remote agent
	successfulRemoteWrites := 0
	for i := range files {
		prepared := files[i].ACHFile

		// Optionally Flatten Batches
		if m.shard.Mergable.FlattenBatches != nil {
			if file, err := prepared.FlattenBatches(); err != nil {
				el.Add(err)
			} else {
				prepared = file
			}
		}

		// Write our file to the mergable directory
		if err := m.saveMergedFile(dir, prepared); err != nil {
			el.Add(fmt.Errorf("problem writing merged file: %v", err))
			logger.Error().Logf("skipping upload of %s after cache failure", prepared)
			continue // skip upload if we can't cache what to upload
		}

		// Upload the file
		if filename, err := f(ctx, i, agent, prepared); err != nil {
			telemetry.RecordError(ctx, err)

			el.Add(fmt.Errorf("problem from callback: %v", err))
		} else {
			files[i].Filename = filename
			successfulRemoteWrites++

			if i > 1 && i%10 == 0 {
				logger.Logf("written (%d/%d) files to remote agent", successfulRemoteWrites, len(files))
			}
		}
	}

	logger.Logf("wrote %d of %d files to remote agent", successfulRemoteWrites, len(files))

	span.SetAttributes(
		attribute.Int("achgateway.successful_remote_writes", successfulRemoteWrites),
	)

	if !el.Empty() {
		return nil, el
	}

	return newProcessedFiles(m.shard.Name, files), nil
}

func makeIndices(total, groups int) []int {
	if groups <= 1 || groups >= total {
		return []int{total}
	}
	xs := []int{0}
	i := 0
	for {
		if i > total {
			break
		}
		i += total / groups
		if i < total {
			xs = append(xs, i)
		}
	}
	return append(xs, total)
}

type mergedFile struct {
	Names    []string
	Filename string
	ACHFile  *ach.File
}

type mergedFiles []mergedFile

func (m *filesystemMerging) chunkFilesTogether(ctx context.Context, indices []int, matches []string, conditions ach.Conditions) (mergedFiles, error) {
	_, span := telemetry.StartSpan(ctx, "chunk-files-together", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.Int("achgateway.indices_num", len(indices)),
		attribute.Int("achgateway.matches", len(matches)),
	))
	defer span.End()

	if len(indices) <= 1 {
		files, err := m.readFiles(matches)
		if err != nil {
			return nil, err
		}
		span.AddEvent("files-read")

		merged, err := ach.MergeFilesWith(files.ACHFiles, conditions)
		if err != nil {
			return nil, err
		}
		span.AddEvent("merged-files")

		out, err := determineMergeDestinations(files, merged), nil
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	var input namedFiles
	mergeParts := make([]*ach.File, 0, len(indices))
	for i := 0; i < len(indices)-1; i += 0 {
		files, err := m.readFiles(matches[indices[i]:indices[i+1]]) // need to keep filename around
		if err != nil {
			return nil, err
		}
		span.AddEvent(fmt.Sprintf("files-read-idx-%d", i))

		input.Names = append(input.Names, files.Names...)
		input.ACHFiles = append(input.ACHFiles, files.ACHFiles...)

		fs, err := ach.MergeFilesWith(files.ACHFiles, conditions)
		if err != nil {
			return nil, err
		}
		span.AddEvent(fmt.Sprintf("merged-files-idx-%d", i))

		i += 1
		mergeParts = append(mergeParts, fs...)
	}

	merged, err := ach.MergeFilesWith(mergeParts, conditions)
	if err != nil {
		return nil, err
	}
	span.AddEvent("final-merge-files")

	out, err := determineMergeDestinations(input, merged), nil
	if err != nil {
		return nil, err
	}
	return out, nil
}

// determineMergeDestinations will compare the input ACH files against the merged files to determine
// where the input files ended up. This allows us to report where an input file was uploaded.
//
// Given we have a list of merged files, for each incoming file we can find it in the merged results.
// This allows us to report which file (and filename) the original file was sent out in.
func determineMergeDestinations(input namedFiles, merged []*ach.File) mergedFiles {
	// Build our result type with merged files
	out := make([]mergedFile, len(merged))
	for i := range merged {
		out[i].ACHFile = merged[i]
	}

	// For each input file find where the EntryDetail landed in the list of merged files
	for i := range input.ACHFiles {
		inputFile := input.ACHFiles[i]

		for j := range out {
			// Find the matching entry
			outFile := out[j].ACHFile

			for ib := range inputFile.Batches {
				ientries := inputFile.Batches[ib].GetEntries()

				for ob := range outFile.Batches {
					oentries := outFile.Batches[ob].GetEntries()

					for ie := range ientries {
						for oe := range oentries {
							if matchingEntry(ientries[ie], oentries[oe]) {
								out[j].Names = append(out[j].Names, input.Names[i])
							}
						}
					}
				}
			}
		}
	}

	// Remove duplicates
	for i := range out {
		slices.Sort(out[i].Names)
		out[i].Names = slices.Compact(out[i].Names)
	}

	return out
}

func matchingEntry(e1, e2 *ach.EntryDetail) bool {
	return e1.TransactionCode == e2.TransactionCode &&
		e1.RDFIIdentification == e2.RDFIIdentification &&
		e1.CheckDigit == e2.CheckDigit &&
		e1.DFIAccountNumber == e2.DFIAccountNumber &&
		e1.Amount == e2.Amount &&
		e1.IdentificationNumber == e2.IdentificationNumber &&
		e1.IndividualName == e2.IndividualName &&
		e1.DiscretionaryData == e2.DiscretionaryData &&
		e1.TraceNumber == e2.TraceNumber
}

func directoryNames(matches []string) []string {
	out := make(map[string]int)
	for i := range matches {
		dir, _ := filepath.Split(matches[i])
		out[dir] += 1
	}
	return maps.Keys(out)
}

func (m *filesystemMerging) saveMergedFile(dir string, file *ach.File) error {
	var buf bytes.Buffer
	if err := ach.NewWriter(&buf).Write(file); err != nil {
		return fmt.Errorf("unable to buffer ACH file: %v", err)
	}

	path := filepath.Join(dir, fmt.Sprintf("%s.ach", hash(buf.Bytes())))

	return m.storage.WriteFile(path, buf.Bytes())
}

func hash(data []byte) string {
	ss := sha256.New()
	ss.Write(data)
	return hex.EncodeToString(ss.Sum(nil))
}
