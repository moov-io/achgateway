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
	"maps"
	"path/filepath"
	"slices"
	"strconv"
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

	// Probe existence, then close before rename. Windows cannot rename a file
	// that still has an open handle.
	originalFile, _ := m.storage.Open(path)
	originalFileWasFound := originalFile != nil
	if originalFile != nil {
		originalFile.Close()
	}

	var canceledFileWasFound bool
	if !originalFileWasFound {
		canceledFile, _ := m.storage.Open(path + ".canceled")
		canceledFileWasFound = canceledFile != nil
		if canceledFile != nil {
			canceledFile.Close()
		}
	}

	// Write the canceled File
	err := m.storage.ReplaceFile(path, path+".canceled")
	if err != nil {
		span.RecordError(err)
	}

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

	// Canceled files are written by HandleCancel as "<path>.canceled" where path
	// is the original ".ach" path (e.g. "dir/foo.ach.canceled"). Older/manual
	// layouts may use "dir/foo.canceled" beside "dir/foo.ach".
	//
	// storage.Glob is non-recursive on a single directory, so every RelativePath
	// shares the same dir prefix — full-stem equality is sufficient and avoids
	// basename false positives across unrelated directories.
	canceledStems := make(map[string]struct{}, len(negativeMatches))
	for j := range negativeMatches {
		stem := strings.TrimSuffix(negativeMatches[j].RelativePath, ".canceled")
		canceledStems[stem] = struct{}{}
	}

	out := make([]string, 0, len(positiveMatches))
	for i := range positiveMatches {
		path := positiveMatches[i].RelativePath
		if _, skip := canceledStems[path]; skip {
			continue
		}
		// "dir/foo.ach" canceled as "dir/foo.canceled"
		if stem, ok := stripACHExtension(path); ok {
			if _, skip := canceledStems[stem]; skip {
				continue
			}
		}
		out = append(out, path)
	}
	return out, nil
}

func (m *filesystemMerging) createMergeConditions() ach.Conditions {
	var mergeConditions ach.Conditions
	if m.shard.Mergable.Conditions != nil {
		mergeConditions = *m.shard.Mergable.Conditions
	}

	// Always cap the MaxDollarAmount at Nacha's file-level debit/credit limit.
	if mergeConditions.MaxDollarAmount == 0 || mergeConditions.MaxDollarAmount > ach.NachaFileDebitCreditLimit {
		mergeConditions.MaxDollarAmount = ach.NachaFileDebitCreditLimit
	}

	return mergeConditions
}

func (m *filesystemMerging) createMergeDirOptions(canceledFiles []string) *ach.MergeDirOptions {
	return &ach.MergeDirOptions{
		AcceptFile:            fileAcceptor(canceledFiles),
		ValidateOptsExtension: ".json",
		FS:                    m.storage,
	}
}

// WithEachMerged does the heavy lifting of coordinating ach.MergeDir, flattening the merged files,
// saving merged files to the local cache, and uploading them to the remote server. After all that
// it generates mappings for what files were put into which uploaded files.
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

	// We need to exclude whatever files have been canceled
	canceledFiles, err := m.getCanceledFiles(ctx, dir)
	if err != nil {
		return nil, fmt.Errorf("problem listing %s for canceled files: %w", dir, err)
	}

	// Snapshot expected inputs before merge/upload work so later local cache
	// writes under uploaded/ cannot affect the FileUploaded completeness check.
	// (storage.Glob is non-recursive; listing early keeps the contract explicit.)
	var el base.ErrorList
	expectedInputs, listErr := m.getNonCanceledMatches(ctx, dir)
	if listErr != nil {
		el.Add(fmt.Errorf("listing input files in %s: %w", dir, listErr))
		expectedInputs = nil
	}

	// Merge the files together
	mergeConditions := m.createMergeConditions()
	opts := m.createMergeDirOptions(canceledFiles)

	files, mergeErr := m.mergeDir(ctx, dir, mergeConditions, opts)
	if mergeErr != nil {
		el.Add(fmt.Errorf("unable to merge files: %w", mergeErr))
	}

	// Only remove an empty isolated directory when merge succeeded and produced
	// nothing. Never delete on merge failure — MergeDir returns nil files with
	// an error, and wiping the dir would destroy every input ACH in the cutoff.
	//
	// Return immediately after a successful empty cleanup so we do not recreate
	// the directory via MkdirAll(uploaded/) below. buildDirMapping runs only
	// after this check (no mapping work / mapErrs on the empty-success path).
	if len(files) == 0 && mergeErr == nil {
		if err := m.storage.RmdirAll(dir); err != nil {
			el.Add(err)
		}
		// Surface pre-merge listing errors (and cleanup failures) even when empty.
		if !el.Empty() {
			return nil, el
		}
		return nil, nil
	}

	// Build entry→input mapping before upload so we can match FileUploaded after.
	// Multiple input files may share an entry identity (same Nacha content under
	// different file_ids) — that is allowed: merge keeps every entry, and the
	// mapping stores a queue of filenames per key so each file_id still gets an event.
	mappings, mapErrs := m.buildDirMapping(dir, canceledFiles)
	for _, mapErr := range mapErrs {
		span.RecordError(mapErr)
		el.Add(mapErr)
	}
	if dupFiles := crossFileDuplicateIdentityFiles(mappings); len(dupFiles) > 0 {
		// Observability only — do not block upload or fail the cutoff.
		span.SetAttributes(
			attribute.Int("achgateway.duplicate_entry_identity_files", len(dupFiles)),
			attribute.StringSlice("achgateway.duplicate_entry_identity_filenames", sampleStrings(dupFiles, 20)),
		)
		logger.Info().Logf("%d input file(s) share duplicate entry identity (all will upload and map): %s",
			len(dupFiles), summarizeFilenames(dupFiles, 20))
	}

	uploadedDir := filepath.Join(dir, "uploaded")
	m.storage.MkdirAll(uploadedDir)

	// Grab our upload Agent
	agent, err := upload.New(m.logger, m.cfg, m.shard.UploadAgent)
	if err != nil {
		return nil, fmt.Errorf("%s merging agent: %v", m.shard.Name, err)
	}
	logger.Logf("found %T agent", agent)

	// Prepare each merged file (flatten / local save). Never return
	// early from this loop — record the error and continue so later files still upload.
	for i := range files {
		// Optionally Flatten Batches. ach.File.FlattenBatches returns a new file
		// on success and does not mutate the receiver on error — keep the original
		// and still upload so money moves; surface the flatten error for ops.
		if m.shard.Mergable.FlattenBatches != nil {
			if file, flatErr := m.flattenBatches(ctx, files[i]); flatErr != nil {
				flatErr = fmt.Errorf("flatten batches for merged file %d: %w", i, flatErr)
				span.RecordError(flatErr)
				el.Add(flatErr)
			} else {
				files[i] = file
			}
		}

		// Best-effort local uploaded/ cache for audit/debug. A save failure must
		// not block the ODFI upload — money movement takes priority; surface the
		// error so operators know the local copy is missing.
		if saveErr := m.saveMergedFile(ctx, uploadedDir, files[i]); saveErr != nil {
			saveErr = fmt.Errorf("writing merged file %d to local cache: %w", i, saveErr)
			span.RecordError(saveErr)
			el.Add(saveErr)
			logger.Error().Logf("local cache save failed for merged file %d; continuing with ODFI upload: %v", i, saveErr)
		}
	}

	// Write each prepared file to the remote agent. Upload failures are recorded
	// but never abort the remaining files in the batch.
	//
	// notUploaded holds ACH files that were prepared but not successfully uploaded
	// (prep skip or upload error). We map their inputs only to exclude them from
	// the unmapped-input error — they already have a more specific error in el
	// and must not receive FileUploaded.
	var merged []mergedFile
	var notUploaded mergedFiles
	successfulRemoteWrites := 0
	eligibleUploads := 0
	for i := range files {
		eligibleUploads++

		filename, upErr := f(ctx, i, agent, files[i]) // upload
		if upErr != nil {
			upErr = fmt.Errorf("problem uploading merged file %d: %w", i, upErr)
			span.RecordError(upErr)
			el.Add(upErr)
			notUploaded = append(notUploaded, mergedFile{ACHFile: files[i]})
			continue
		}

		merged = append(merged, mergedFile{
			UploadedFilename: filename,
			ACHFile:          files[i],
			Shard:            m.shard.Name,
		})
		successfulRemoteWrites++

		if successfulRemoteWrites > 1 && successfulRemoteWrites%10 == 0 {
			logger.Logf("written (%d/%d) files to remote agent", successfulRemoteWrites, eligibleUploads)
		}
	}
	logger.Logf("wrote %d of %d files to remote agent", successfulRemoteWrites, eligibleUploads)

	span.SetAttributes(
		attribute.Int("achgateway.successful_remote_writes", successfulRemoteWrites),
	)

	// From the pre-upload mapping match each input against merged/uploaded files.
	// Clone because findInputFilepaths deletes keys as it matches.
	var leftoverEntryKeys int
	if mappings != nil {
		merged, leftoverEntryKeys = m.findInputFilepaths(maps.Clone(mappings), merged)
		// Account for not-uploaded outputs so their inputs are not double-reported
		// as "unmapped" when prep/upload errors already explain the miss.
		if len(notUploaded) > 0 {
			notUploaded, _ = m.findInputFilepaths(maps.Clone(mappings), notUploaded)
		}
	}

	// Completeness: inputs not present on any successful upload AND not explained
	// by a prep/upload failure on their merged output.
	accounted := make(mergedFiles, 0, len(merged)+len(notUploaded))
	accounted = append(accounted, merged...)
	accounted = append(accounted, notUploaded...)
	unmapped := missingMappedInputFiles(expectedInputs, accounted)
	mappedCount := countMappedInputFiles(merged)

	span.SetAttributes(
		attribute.Int("achgateway.expected_input_files", len(expectedInputs)),
		attribute.Int("achgateway.mapped_input_files", mappedCount),
		attribute.Int("achgateway.unmapped_input_files", len(unmapped)),
		// leftoverEntryKeys is remaining identity-queue slots (not distinct file IDs).
		attribute.Int("achgateway.unmapped_entry_keys", leftoverEntryKeys),
	)
	// Only flag genuine mapping gaps when money moved. Total prep/upload failure
	// already surfaces via those errors without an unmapped double-count.
	if len(unmapped) > 0 && successfulRemoteWrites > 0 {
		sample := unmapped
		if len(sample) > 20 {
			sample = sample[:20]
		}
		span.SetAttributes(attribute.StringSlice("achgateway.unmapped_filenames", sample))
		recordUnmappedInputFiles(m.shard.Name, len(unmapped))

		unmappedErr := fmt.Errorf("%d input file(s) in %s were not mapped to an uploaded file (FileUploaded will not be emitted for them): %s",
			len(unmapped), dir, summarizeFilenames(unmapped, 20))
		logger.Error().LogError(unmappedErr)
		span.RecordError(unmappedErr)
		el.Add(unmappedErr)
	} else if len(unmapped) > 0 {
		span.SetAttributes(attribute.StringSlice("achgateway.unmapped_filenames", sampleStrings(unmapped, 20)))
	}

	if el.Empty() {
		return merged, nil
	}
	return merged, el
}

func (m *filesystemMerging) mergeDir(ctx context.Context, dir string, mergeConditions ach.Conditions, opts *ach.MergeDirOptions) ([]*ach.File, error) {
	_, span := telemetry.StartSpan(ctx, "ach-merge-dir", trace.WithAttributes(
		attribute.String("achgateway.shard", m.shard.Name),
		attribute.String("achgateway.dir", dir),
		attribute.Int("achgateway.merge_max_lines", mergeConditions.MaxLines),
		attribute.Int64("achgateway.merge_max_dollar_amount", mergeConditions.MaxDollarAmount),
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

// fileAcceptor builds a MergeDir AcceptFile callback.
// canceledFiles must be basenames from getCanceledFiles (e.g. "foo.ach"), already
// stripped of the ".canceled" suffix — matched against filepath.Base of each walk path.
func fileAcceptor(canceledFiles []string) func(string) ach.FileAcceptance {
	// O(1) lookup instead of slices.Contains on every path during MergeDir walks.
	canceled := make(map[string]struct{}, len(canceledFiles))
	for _, name := range canceledFiles {
		// Index basename only — getCanceledFiles always returns basenames.
		canceled[filepath.Base(name)] = struct{}{}
	}

	return func(path string) ach.FileAcceptance {
		// Reject canceled files
		if strings.Contains(path, ".canceled") {
			return ach.SkipFile
		}
		_, filename := filepath.Split(path)
		if _, skip := canceled[filename]; skip {
			return ach.SkipFile
		}

		// Only accept .ach files (case-insensitive, allocation-free suffix check)
		if hasACHExtension(path) {
			return ach.AcceptFile
		}
		return ach.SkipFile
	}
}

// hasACHExtension reports whether path ends with ".ach" ignoring case.
func hasACHExtension(path string) bool {
	return len(path) >= 4 && strings.EqualFold(path[len(path)-4:], ".ach")
}

// stripACHExtension returns path without a trailing ".ach"/".ACH" suffix.
func stripACHExtension(path string) (string, bool) {
	if !hasACHExtension(path) {
		return path, false
	}
	return path[:len(path)-4], true
}

// trimACHExtension returns path without a trailing ".ach"/".ACH" suffix, or
// path unchanged when the suffix is absent.
func trimACHExtension(path string) string {
	if stem, ok := stripACHExtension(path); ok {
		return stem
	}
	return path
}

// entryFileMapping maps entry identity keys to the input filename(s) they came from.
// A key may map to multiple filenames when distinct file_ids carry identical ACH
// entry content — moov-io/ach keeps each entry, and we must emit FileUploaded for each.
type entryFileMapping map[string][]string

// buildDirMapping computes a map of the input files and their entries together so that we can quickly find
// where they were merged into.
//
// Errors reading individual input files are collected and returned without aborting the walk —
// remaining files still contribute to the mapping so FileUploaded can be emitted for them.
// A directory-level ReadDir failure returns a nil map and a single error.
func (m *filesystemMerging) buildDirMapping(dir string, canceledFiles []string) (entryFileMapping, []error) {
	mappings := make(entryFileMapping)
	// seen[identityKey][filename] for O(1) dedupe while building queues.
	seen := make(map[string]map[string]struct{})

	fds, err := m.storage.ReadDir(dir)
	if err != nil {
		return nil, []error{err}
	}

	acceptor := fileAcceptor(canceledFiles)
	var errs []error

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

		fullPath := filepath.Join(dir, path)
		if err := m.accumulateMappings(mappings, seen, fullPath); err != nil {
			errs = append(errs, fmt.Errorf("accumulating mappings from %s failed: %w", path, err))
			// Continue so one bad input does not block FileUploaded for the rest.
			continue
		}
	}

	return mappings, errs
}

func (m *filesystemMerging) accumulateMappings(mappings entryFileMapping, seen map[string]map[string]struct{}, path string) error {
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

	// Queue each BatchHeader+Entry under its identity key.
	//
	//   - Same entry under different file_ids → each file_id is queued (upload
	//     each entry; FileUploaded for each file_id).
	//   - Same file_id already queued for this key → skip (idempotent re-queue /
	//     duplicate rows in one file only need one FileUploaded for that file_id).
	for i := range file.Batches {
		bh := file.Batches[i].GetHeader()

		entries := file.Batches[i].GetEntries()

		for idx := range entries {
			if entries[idx] == nil {
				continue
			}
			key := makeKey(bh, entries[idx])
			if seen[key] == nil {
				seen[key] = make(map[string]struct{})
			}
			if _, exists := seen[key][filename]; exists {
				continue
			}
			seen[key][filename] = struct{}{}
			mappings[key] = append(mappings[key], filename)
		}
	}

	return nil
}

// crossFileDuplicateIdentityFiles returns sorted unique basenames that share an
// entry identity key with at least one other distinct input file.
func crossFileDuplicateIdentityFiles(mappings entryFileMapping) []string {
	if len(mappings) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	for _, names := range mappings {
		uniq := uniqueSortedStrings(names)
		if len(uniq) < 2 {
			continue
		}
		for _, name := range uniq {
			seen[name] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

func uniqueSortedStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := slices.Clone(in)
	slices.Sort(out)
	return slices.Compact(out)
}

func mappingLeftoverCount(mappings entryFileMapping) int {
	n := 0
	for _, names := range mappings {
		n += len(names)
	}
	return n
}

func makeKey(bh *ach.BatchHeader, entry *ach.EntryDetail) string {
	// Callers skip nil entries. Do not return "" — that would collide in the mapping.
	if entry == nil {
		return "\x00nil-entry"
	}

	// MergeDir combines batches with BatchHeader.Equal, then copies entries under
	// the first matching header. Equal ignores CompanyDiscretionaryData,
	// CompanyDescriptiveDate, SettlementDate, OriginatorStatusCode, and company
	// name case — all of which appear in BatchHeader.String()[:87] (and BatchNumber,
	// which merge also rewrites). Keys built from the original 87-char header
	// therefore miss after merge, so FileUploaded is not emitted for those inputs.
	var b strings.Builder
	if bh != nil {
		b.Grow(96 + 94)
		b.WriteString(strconv.Itoa(bh.ServiceClassCode))
		b.WriteByte('|')
		b.WriteString(strings.ToUpper(bh.CompanyName))
		b.WriteByte('|')
		b.WriteString(bh.CompanyIdentification)
		b.WriteByte('|')
		b.WriteString(bh.StandardEntryClassCode)
		b.WriteByte('|')
		b.WriteString(bh.CompanyEntryDescription)
		b.WriteByte('|')
		b.WriteString(bh.EffectiveEntryDate)
		b.WriteByte('|')
		b.WriteString(bh.ODFIIdentification)
		b.WriteByte('|')
	}
	b.WriteString(entry.String())
	return b.String()
}

// mergedFile represents a single merged file structure used in the merging and uploading process.
type mergedFile struct {
	InputFilepaths   []string
	UploadedFilename string
	ACHFile          *ach.File
	Shard            string
}

type mergedFiles []mergedFile

// findInputFilepaths matches merged ACH entries back to their input filenames.
// It returns the merged files with InputFilepaths populated and the number of
// leftover mapping slots (input entry identities not matched — typically none
// after a full merge).
//
// When several input files share one entry identity, each matching merged entry
// consumes one queued filename so every file_id can receive FileUploaded.
func (m *filesystemMerging) findInputFilepaths(mappings entryFileMapping, merged mergedFiles) (mergedFiles, int) {
	// Compare each merged file against mappings
	for i := range merged {
		if merged[i].ACHFile == nil {
			continue
		}
		for j := range merged[i].ACHFile.Batches {
			batch := merged[i].ACHFile.Batches[j]

			bh := batch.GetHeader()
			entries := batch.GetEntries()

			for idx := range entries {
				if entries[idx] == nil {
					continue
				}
				key := makeKey(bh, entries[idx])

				names := mappings[key]
				if len(names) == 0 {
					continue
				}
				merged[i].InputFilepaths = append(merged[i].InputFilepaths, names[0])
				if len(names) == 1 {
					delete(mappings, key)
				} else {
					mappings[key] = names[1:]
				}
			}
		}

		// Compact duplicate basenames from multi-entry inputs in one merged file.
		// Safe: isolateMergableDir is flat (no nested ACH dirs), and accumulateMappings
		// stores filepath.Base only — two distinct inputs cannot share a basename.
		slices.Sort(merged[i].InputFilepaths)
		merged[i].InputFilepaths = slices.Compact(merged[i].InputFilepaths)
	}
	return merged, mappingLeftoverCount(mappings)
}

// missingMappedInputFiles returns base filenames from expectedInputs that do not
// appear in any merged file's InputFilepaths. expectedInputs may be relative paths.
func missingMappedInputFiles(expectedInputs []string, merged mergedFiles) []string {
	if len(expectedInputs) == 0 {
		return nil
	}

	mapped := make(map[string]struct{}, len(expectedInputs))
	for i := range merged {
		for _, p := range merged[i].InputFilepaths {
			mapped[filepath.Base(p)] = struct{}{}
		}
	}

	var missing []string
	seen := make(map[string]struct{}, len(expectedInputs))
	for _, p := range expectedInputs {
		name := filepath.Base(p)
		if name == "" || name == "." || name == string(filepath.Separator) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if _, ok := mapped[name]; !ok {
			missing = append(missing, name)
		}
	}
	slices.Sort(missing)
	return missing
}

func countMappedInputFiles(merged mergedFiles) int {
	seen := make(map[string]struct{})
	for i := range merged {
		for _, p := range merged[i].InputFilepaths {
			seen[filepath.Base(p)] = struct{}{}
		}
	}
	return len(seen)
}

func summarizeFilenames(names []string, limit int) string {
	if len(names) == 0 {
		return ""
	}
	if limit <= 0 || len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s, ... (+%d more)", strings.Join(names[:limit], ", "), len(names)-limit)
}

func sampleStrings(names []string, limit int) []string {
	if limit <= 0 || len(names) <= limit {
		return names
	}
	return names[:limit]
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
