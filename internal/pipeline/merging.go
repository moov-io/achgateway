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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
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
	HandleXfer(xfer incoming.ACHFile) error
	HandleCancel(cancel incoming.CancelACHFile) error

	WithEachMerged(f func(int, upload.Agent, *ach.File) error) (*processedFiles, error)
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

func (m *filesystemMerging) HandleXfer(xfer incoming.ACHFile) error {
	if err := m.writeACHFile(xfer); err != nil {
		return m.logger.LogErrorf("problem writing ACH file: %v", err).Err()
	}
	return nil
}

func (m *filesystemMerging) writeACHFile(xfer incoming.ACHFile) error {
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

func (m *filesystemMerging) HandleCancel(cancel incoming.CancelACHFile) error {
	path := filepath.Join("mergable", m.shard.Name, fmt.Sprintf("%s.ach", cancel.FileID))

	// Write the canceled File
	return m.storage.ReplaceFile(path, path+".canceled")
}

func (m *filesystemMerging) isolateMergableDir() (string, error) {
	newdir := filepath.Join(fmt.Sprintf("%s-%v", m.shard.Name, time.Now().Format("20060102-150405")))

	// Otherwise attempt to isolate the directory
	return newdir, m.storage.ReplaceDir(filepath.Join("mergable", m.shard.Name), newdir)
}

func (m *filesystemMerging) getNonCanceledMatches(path string) ([]string, error) {
	positiveMatches, err := m.storage.Glob(path + "/*.ach")
	if err != nil {
		return nil, err
	}
	negativeMatches, err := m.storage.Glob(path + "/*.canceled")
	if err != nil {
		return nil, err
	}

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
	fileIDs  []string
}

func newProcessedFiles(shardKey string, matches []string) *processedFiles {
	processed := &processedFiles{shardKey: shardKey}

	for i := range matches {
		// each match follows $path/$fileID.ach
		fileID := strings.TrimSuffix(filepath.Base(matches[i]), ".ach")
		processed.fileIDs = append(processed.fileIDs, fileID)
	}

	return processed
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

func (m *filesystemMerging) readFiles(paths []string) ([]*ach.File, error) {
	var out []*ach.File
	for i := range paths {
		file, err := m.readFile(paths[i])
		if err != nil {
			return nil, fmt.Errorf("reading %s failed: %w", paths[i], err)
		}
		out = append(out, file)
	}
	return out, nil
}

func (m *filesystemMerging) WithEachMerged(f func(int, upload.Agent, *ach.File) error) (*processedFiles, error) {
	processed := &processedFiles{}

	// move the current directory so it's isolated and easier to debug later on
	dir, err := m.isolateMergableDir()
	if err != nil {
		return nil, fmt.Errorf("problem isolating newdir=%s error=%v", dir, err)
	}

	matches, err := m.getNonCanceledMatches(dir)
	if err != nil {
		return nil, fmt.Errorf("problem with %s glob: %v", dir, err)
	}

	logger := m.logger.Set("shardName", log.String(m.shard.Name))

	dirNames := strings.Join(directoryNames(matches), ", ")
	logger.Logf("found %d matching ACH files in %v", len(matches), dirNames)

	var el base.ErrorList

	// Merge files together in groups
	// TODO(adam): Make the group size configurable
	var mergeConditions ach.Conditions
	if m.shard.Mergable.Conditions != nil {
		mergeConditions = *m.shard.Mergable.Conditions
	}

	indices := makeIndices(len(matches), len(matches)/100)
	files, err := m.chunkFilesTogether(indices, matches, mergeConditions)
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
		// Optionally Flatten Batches
		if m.shard.Mergable.FlattenBatches != nil {
			if file, err := files[i].FlattenBatches(); err != nil {
				el.Add(err)
			} else {
				files[i] = file
			}
		}

		// Write our file to the mergable directory
		if err := m.saveMergedFile(dir, files[i]); err != nil {
			el.Add(fmt.Errorf("problem writing merged file: %v", err))
			logger.Error().Logf("skipping upload of %s after cache failure", files[i])
			continue // skip upload if we can't cache what to upload
		}

		// Upload the file
		if err := f(i, agent, files[i]); err != nil {
			el.Add(fmt.Errorf("problem from callback: %v", err))
		} else {
			successfulRemoteWrites++

			if i > 1 && i%10 == 0 {
				logger.Logf("written (%d/%d) files to remote agent", successfulRemoteWrites, len(files))
			}
		}
	}

	logger.Logf("wrote %d of %d files to remote agent", successfulRemoteWrites, len(files))

	if !el.Empty() {
		return nil, el
	}

	return newProcessedFiles(m.shard.Name, matches), nil
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

func (m *filesystemMerging) chunkFilesTogether(indices []int, matches []string, conditions ach.Conditions) ([]*ach.File, error) {
	if len(indices) <= 1 {
		files, err := m.readFiles(matches)
		if err != nil {
			return nil, err
		}
		return ach.MergeFilesWith(files, conditions)
	}

	var out []*ach.File
	for i := 0; i < len(indices)-1; i += 0 {
		files, err := m.readFiles(matches[indices[i]:indices[i+1]])
		if err != nil {
			return nil, err
		}
		fs, err := ach.MergeFilesWith(files, conditions)
		if err != nil {
			return nil, err
		}
		i += 1
		out = append(out, fs...)
	}
	return ach.MergeFilesWith(out, conditions)
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
