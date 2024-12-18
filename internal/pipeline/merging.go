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
	"io/fs"
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

	"github.com/igrmk/treemap/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	HandleCancel(ctx context.Context, cancel incoming.CancelACHFile) (incoming.FileCancellationResponse, error)

	WithEachMerged(ctx context.Context, f func(context.Context, int, upload.Agent, *ach.File) (string, error)) (mergedFiles, error)
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

	fileID := strings.TrimSuffix(xfer.FileID, ".ach")
	path := filepath.Join("mergable", m.shard.Name, fileID+".ach")
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

		path := filepath.Join("mergable", m.shard.Name, fileID+".json")
		if err := m.storage.WriteFile(path, buf.Bytes()); err != nil {
			m.logger.Warn().With(log.Fields{
				"fileID":   log.String(xfer.FileID),
				"shardKey": log.String(xfer.ShardKey),
			}).Logf("ERROR writing ValidateOpts: %v", err)
		}
	}

	return nil
}

func (m *filesystemMerging) HandleCancel(ctx context.Context, cancel incoming.CancelACHFile) (incoming.FileCancellationResponse, error) {
	_, span := telemetry.StartSpan(ctx, "handle-cancel", trace.WithAttributes(
		attribute.String("achgateway.file_id", cancel.FileID),
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.String("achgateway.shard_key", cancel.ShardKey),
	))
	defer span.End()

	fileID := strings.TrimSuffix(cancel.FileID, ".ach")
	path := filepath.Join("mergable", m.shard.Name, fileID+".ach")

	// Check if the file exists already
	originalFile, _ := m.storage.Open(path)
	if originalFile != nil {
		defer originalFile.Close()
	}

	// Check if the canceled file exists already
	var canceledFile fs.File
	if originalFile == nil {
		canceledFile, _ = m.storage.Open(path + ".canceled")
		if canceledFile != nil {
			defer canceledFile.Close()
		}
	}

	// Write the canceled File
	err := m.storage.ReplaceFile(path, path+".canceled")
	if err != nil {
		span.RecordError(err)
	}

	originalFileWasFound := originalFile != nil
	canceledFileWasFound := canceledFile != nil
	successfulReplace := err == nil

	span.SetAttributes(
		attribute.Bool("achgateway.canceled_file_found", canceledFileWasFound),
		attribute.Bool("achgateway.cancel_replacement_written", successfulReplace),
		attribute.Bool("achgateway.original_file_found", originalFileWasFound),
		attribute.String("achgateway.path", path),
	)

	// We need a file to be found and we no errors during the rename
	var successful bool = (originalFileWasFound || canceledFileWasFound) && successfulReplace

	out := incoming.FileCancellationResponse{
		FileID:     cancel.FileID,
		ShardKey:   cancel.ShardKey,
		Successful: successful,
	}
	return out, err
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

func (m *filesystemMerging) getCanceledFiles(ctx context.Context, dir string) ([]string, error) {
	_, span := telemetry.StartSpan(ctx, "get-canceled-files", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.String("achgateway.dir", dir),
	))
	defer span.End()

	matches, err := m.storage.Glob(dir + "/*.canceled")
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.Int("achgateway.canceled_files", len(matches)))

	out := make([]string, len(matches))
	for i := range matches {
		_, filename := filepath.Split(matches[i].RelativePath)
		out[i] = strings.TrimSuffix(filename, ".canceled")
	}
	return out, nil
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

func (m *filesystemMerging) createMergeDirOptions(canceledFiles []string) *ach.MergeDirOptions {
	return &ach.MergeDirOptions{
		AcceptFile:            fileAcceptor(canceledFiles),
		ValidateOptsExtension: ".json",
		FS:                    m.storage,
	}
}

func (m *filesystemMerging) WithEachMerged(ctx context.Context, f func(context.Context, int, upload.Agent, *ach.File) (string, error)) (mergedFiles, error) {
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

	logger := m.logger.Set("shardName", log.String(m.shard.Name))

	// Merge the files together
	var mergeConditions ach.Conditions
	if m.shard.Mergable.Conditions != nil {
		mergeConditions = *m.shard.Mergable.Conditions
	}

	canceledFiles, err := m.getCanceledFiles(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("problem listing %s for canceled files: %w", dir, err)
	}

	var el base.ErrorList

	opts := m.createMergeDirOptions(canceledFiles)
	files, err := m.mergeDir(ctx, dir, mergeConditions, opts)
	if err != nil {
		el.Add(fmt.Errorf("unable to merge files: %v", err))
	}

	// Remove the directory if there are no files, otherwise setup an inner dir for the uploaded file.
	if len(files) == 0 {
		// delete the new directory as there's nothing to merge
		if err := m.storage.RmdirAll(dir); err != nil {
			el.Add(err)
		}
	}

	uploadedDir := filepath.Join(dir, "uploaded")
	m.storage.MkdirAll(uploadedDir)

	// Grab our upload Agent
	agent, err := upload.New(m.logger, m.cfg, m.shard.UploadAgent)
	if err != nil {
		return nil, fmt.Errorf("%s merging agent: %v", m.shard.Name, err)
	}
	logger.Logf("found %T agent", agent)

	// Write each file to local cache
	for i := range files {
		// Optionally Flatten Batches
		if m.shard.Mergable.FlattenBatches != nil {
			if file, err := m.flattenBatches(ctx, files[i]); err != nil {
				el.Add(err)
			} else {
				files[i] = file
			}
		}

		// Write our file to the mergable directory
		if err := m.saveMergedFile(ctx, uploadedDir, files[i]); err != nil {
			err = fmt.Errorf("problem writing merged file: %v", err)
			span.RecordError(err)
			el.Add(err)
			continue
		}
	}

	// Write each file to the remote agent
	var merged []mergedFile
	successfulRemoteWrites := 0
	for i := range files {
		filename, err := f(ctx, i, agent, files[i]) // upload
		if err != nil {
			err = fmt.Errorf("problem from callback: %v", err)
			span.RecordError(err)
			el.Add(err)
		} else {
			merged = append(merged, mergedFile{
				UploadedFilename: filename,
				ACHFile:          files[i],
				Shard:            m.shard.Name,
			})
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

	// Build a mapping of BatchHeader + EntryDetail from dir (input files)
	mappings, err := m.buildDirMapping(dir, canceledFiles)
	if err != nil {
		el.Add(err)
	}

	// From that mapping match each one against the merged/uploaded files
	merged = m.findInputFilepaths(mappings, merged)

	if el.Empty() {
		return merged, nil
	}
	return merged, el
}

func (m *filesystemMerging) mergeDir(ctx context.Context, dir string, mergeConditions ach.Conditions, opts *ach.MergeDirOptions) ([]*ach.File, error) {
	_, span := telemetry.StartSpan(ctx, "ach-merge-dir", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.String("achgateway.dir", dir),
	))
	defer span.End()

	files, err := ach.MergeDir(dir, mergeConditions, opts)
	if err != nil {
		span.RecordError(err)
	}

	span.SetAttributes(attribute.Int("achgateway.merged_files", len(files)))

	return files, err
}

func (m *filesystemMerging) flattenBatches(ctx context.Context, file *ach.File) (*ach.File, error) {
	_, span := telemetry.StartSpan(ctx, "ach-flatten-batches", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
	))
	defer span.End()

	return file.FlattenBatches()
}

func fileAcceptor(canceledFiles []string) func(string) ach.FileAcceptance {
	return func(path string) ach.FileAcceptance {
		// Reject canceled files
		if strings.Contains(path, ".canceled") {
			return ach.SkipFile
		}
		_, filename := filepath.Split(path)
		if slices.Contains(canceledFiles, filename) {
			return ach.SkipFile
		}

		// Only accept .ach files
		if strings.Contains(path, ".ach") {
			return ach.AcceptFile
		}
		return ach.SkipFile
	}
}

// buildDirMapping computes a tree of the input files and their entries together so that we can quickly find
// where they were merged into.
func (m *filesystemMerging) buildDirMapping(dir string, canceledFiles []string) (*treemap.TreeMap[string, string], error) {
	tree := treemap.New[string, string]()

	fds, err := m.storage.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	acceptor := fileAcceptor(canceledFiles)

	for i := range fds {
		path := fds[i].Name()

		// Ignore directories as ReadDir continues inside of them.
		// .json files contain ValidateOpts which we can skip
		if fds[i].IsDir() || strings.HasSuffix(path, ".json") {
			continue
		}

		// Skip the file if merging would have skipped it
		if acceptor(path) == ach.SkipFile {
			continue
		}

		err = m.accumulateMappings(tree, filepath.Join(dir, path))
		if err != nil {
			return nil, fmt.Errorf("accumulating mappings from %s failed: %w", path, err)
		}
	}

	return tree, nil
}

func (m *filesystemMerging) accumulateMappings(tree *treemap.TreeMap[string, string], path string) error {
	fd, err := m.storage.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s failed: %w", path, err)
	}
	defer fd.Close()

	// Check for validate opts
	validateOptsPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".json"
	var validateOpts *ach.ValidateOpts
	if optsFD, err := m.storage.Open(validateOptsPath); err == nil {
		if optsFD != nil {
			defer optsFD.Close()
		}

		err = json.NewDecoder(optsFD).Decode(&validateOpts)
		if err != nil {
			return fmt.Errorf("reading %s as validate opts failed: %w", validateOptsPath, err)
		}
	}

	rdr := ach.NewReader(fd)
	if validateOpts != nil {
		rdr.SetValidation(validateOpts)
	}

	file, err := rdr.Read()
	if err != nil {
		return fmt.Errorf("reading %s failed: %w", path, err)
	}

	_, filename := filepath.Split(path)

	// Add each BatchHeader and Entry to the map
	for i := range file.Batches {
		bh := file.Batches[i].GetHeader().String()

		entries := file.Batches[i].GetEntries()

		for m := range entries {
			key := makeKey(bh, entries[m])
			tree.Set(key, filename)
		}
	}

	return nil
}

func makeKey(bh string, entry *ach.EntryDetail) string {
	// copy off the BatchNumber from our header, it's modified when merging
	return fmt.Sprintf("%87.87s%s", bh, entry.String())
}

type mergedFile struct {
	InputFilepaths   []string
	UploadedFilename string
	ACHFile          *ach.File
	Shard            string
}

type mergedFiles []mergedFile

func (m *filesystemMerging) findInputFilepaths(mappings *treemap.TreeMap[string, string], merged mergedFiles) mergedFiles {
	// Compare each merged file against mappings
	for i := range merged {
		for j := range merged[i].ACHFile.Batches {
			batch := merged[i].ACHFile.Batches[j]

			bh := batch.GetHeader().String()
			entries := batch.GetEntries()

			for m := range entries {
				key := makeKey(bh, entries[m])

				filename, found := mappings.Get(key)
				if found {
					merged[i].InputFilepaths = append(merged[i].InputFilepaths, filename)
					mappings.Del(key)
				}
			}
		}

		slices.Sort(merged[i].InputFilepaths)
		merged[i].InputFilepaths = slices.Compact(merged[i].InputFilepaths)
	}
	return merged
}

func (m *filesystemMerging) saveMergedFile(ctx context.Context, dir string, file *ach.File) error {
	_, span := telemetry.StartSpan(ctx, "saved-merged-file", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.String("achgateway.dir", dir),
	))
	defer span.End()

	var buf bytes.Buffer
	err := ach.NewWriter(&buf).Write(file)
	if err != nil {
		return fmt.Errorf("unable to buffer ACH file: %w", err)
	}

	name := hash(buf.Bytes())
	path := filepath.Join(dir, name+".ach")

	span.SetAttributes(
		attribute.String("achgateway.merged_filename", path),
		attribute.Int("achgateway.merged_filesize_bytes", buf.Len()),
	)

	err = m.storage.WriteFile(path, buf.Bytes())
	if err != nil {
		return fmt.Errorf("writing merged ACH file: %w", err)
	}

	validateOpts := file.GetValidation()
	if validateOpts != nil {
		buf.Reset()

		err = json.NewEncoder(&buf).Encode(validateOpts)
		if err != nil {
			return fmt.Errorf("marshal of merged ACH file validate opts: %w", err)
		}

		path = filepath.Join(dir, name+".json")
		err = m.storage.WriteFile(path, buf.Bytes())
		if err != nil {
			return fmt.Errorf("writing merged ACH file validate opts: %w", err)
		}
	}

	return nil
}

func hash(data []byte) string {
	ss := sha256.New()
	ss.Write(data)
	return hex.EncodeToString(ss.Sum(nil))
}
