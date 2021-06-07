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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/consul"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/strx"
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

	WithEachMerged(f func(upload.Agent, *ach.File) error) (*processedTransfers, error)
}

func NewMerging(logger log.Logger, consul *consul.Wrapper, shard service.Shard, cfg service.UploadAgents) (XferMerging, error) {
	dir := filepath.Join("storage", "mergable") // default directory
	if cfg.Merging.Directory != "" {
		dir = filepath.Join(cfg.Merging.Directory, "mergable")
	}

	if err := os.MkdirAll(dir, 0777); err != nil {
		return nil, err
	}

	return &filesystemMerging{
		baseDir: dir,
		logger:  logger,
		cfg:     cfg,
		shard:   shard,
		consul:  consul,
	}, nil
}

type filesystemMerging struct {
	logger  log.Logger
	baseDir string
	cfg     service.UploadAgents
	shard   service.Shard
	consul  *consul.Wrapper
}

func (m *filesystemMerging) HandleXfer(xfer incoming.ACHFile) error {
	if err := m.writeACHFile(xfer); err != nil {
		return m.logger.LogErrorf("problem writing ACH file: %v", err).Err()
	}
	return nil
}

func (m *filesystemMerging) writeACHFile(xfer incoming.ACHFile) error {
	var buf bytes.Buffer
	if err := ach.NewWriter(&buf).Write(xfer.File); err != nil {
		return err
	}

	path := filepath.Join(m.baseDir, xfer.ShardKey)
	os.MkdirAll(path, 0777)

	path = filepath.Join(path, fmt.Sprintf("%s.ach", xfer.FileID))
	if err := ioutil.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}

func (m *filesystemMerging) HandleCancel(cancel incoming.CancelACHFile) error {
	path := filepath.Join(m.baseDir, cancel.ShardKey)
	os.MkdirAll(path, 0777)

	// Write the canceled file
	path = filepath.Join(path, fmt.Sprintf("%s.ach", cancel.FileID))
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		// file doesn't exist, so write one
		return ioutil.WriteFile(path+".canceled", nil, 0644)
	} else {
		// move the existing file
		return os.Rename(path, path+".canceled")
	}
}

func (m *filesystemMerging) isolateMergableDir() (string, error) {
	// rename m.baseDir so we're the only accessor for it, then recreate m.baseDir
	parent, _ := filepath.Split(m.baseDir)
	newdir := filepath.Join(parent, time.Now().Format("20060102-150405"))
	if err := os.Rename(m.baseDir, newdir); err != nil {
		return newdir, err
	}
	return newdir, os.Mkdir(m.baseDir, 0777) // create m.baseDir again
}

func getNonCanceledMatches(path string) ([]string, error) {
	positiveMatches, err := filepath.Glob(path)
	if err != nil {
		return nil, err
	}
	negativeMatches, err := filepath.Glob(path + "*.canceled")
	if err != nil {
		return nil, err
	}

	var out []string
	for i := range positiveMatches {
		exclude := false
		for j := range negativeMatches {
			// We match when a "XXX.ach.canceled" filepath exists and so we can't
			// include "XXX.ach" has a filepath from this function.
			if strings.HasPrefix(negativeMatches[j], positiveMatches[i]) {
				exclude = true
				break
			}
		}
		if !exclude {
			out = append(out, positiveMatches[i])
		}
	}
	return out, nil
}

type processedTransfers struct {
	transferIDs []string
}

func newProcessedTransfers(matches []string) *processedTransfers {
	processed := &processedTransfers{}

	for i := range matches {
		// each match follows $path/$transferID.ach so we can split that
		// and grab the transferID
		transferID := strings.TrimSuffix(filepath.Base(matches[i]), ".ach")
		processed.transferIDs = append(processed.transferIDs, transferID)
	}

	return processed
}

func (m *filesystemMerging) WithEachMerged(f func(upload.Agent, *ach.File) error) (*processedTransfers, error) {
	// move the current directory so it's isolated and easier to debug later on
	dir, err := m.isolateMergableDir()
	if err != nil {
		return nil, fmt.Errorf("problem isolating newdir=%s error=%v", dir, err)
	}
	// Grab each tenant directory and process it
	dirs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("problem reading tenant dirs: %v", err)
	}

	// Process each Organization directory
	processed := &processedTransfers{}
	for i := range dirs {
		if dirs[i].IsDir() {
			shardKey := filepath.Base(dirs[i].Name())
			logger := m.logger.Set("shardKey", log.String(shardKey))
			logger.Logf("processing mergable directory %s", dirs[i].Name())

			// Grab our upload Agent
			agent, err := upload.New(m.logger, m.cfg, strx.Or(m.shard.UploadAgent, m.cfg.DefaultAgentID))
			if err != nil {
				return processed, fmt.Errorf("agent: %v", err)
			}
			logger.Logf("found %T agent", agent)

			// Hand off the directory to be merged and uploaded
			proc, err := m.withEachOrganizationDirectory(filepath.Join(dir, dirs[i].Name()), shardKey, agent, f)
			if err != nil {
				return processed, fmt.Errorf("each tenant (%s): %v", filepath.Base(dirs[i].Name()), err)
			}

			logger.Logf("processed %d transfers: %#v", len(proc.transferIDs), proc.transferIDs)
			processed.transferIDs = append(processed.transferIDs, proc.transferIDs...)
		} else {
			m.logger.Logf("unexpected file info: %v", dirs[i].Name())
		}
	}
	return processed, nil
}

func (m *filesystemMerging) withEachOrganizationDirectory(dir string, shardKey string, agent upload.Agent, f func(upload.Agent, *ach.File) error) (*processedTransfers, error) {
	path := filepath.Join(dir, "*.ach")
	matches, err := getNonCanceledMatches(path)
	if err != nil {
		return nil, fmt.Errorf("problem with %s glob: %v", path, err)
	}

	tenantID := filepath.Base(dir)
	logger := m.logger.Set("tenantID", log.String(tenantID))
	logger.Logf("found %d matching ACH files: %#v", len(matches), matches)

	var files []*ach.File
	var el base.ErrorList
	for i := range matches {
		file, err := ach.ReadFile(matches[i])
		if err != nil {
			el.Add(fmt.Errorf("problem reading %s: %v", matches[i], err))
			continue
		}
		if file != nil {
			files = append(files, file)
		}
	}

	// Combine Batches into one file, force ascending TraceNumbers starting from the first EntryDetail.
	files, err = ach.MergeFiles(files)
	if err != nil {
		el.Add(fmt.Errorf("unable to merge files: %v", err))
	}

	if len(matches) > 0 {
		logger.Logf("merged %d transfers into %d files", len(matches), len(files))
	}

	// Remove the directory if there are no files, otherwise setup an inner dir for the uploaded file.
	if len(files) == 0 {
		// delete the new directory as there's nothing to merge
		if err := os.RemoveAll(dir); err != nil {
			el.Add(err)
		}
	} else {
		dir = filepath.Join(dir, "uploaded")
		os.MkdirAll(dir, 0777)
	}

	// Write each file to our remote agent
	successfulRemoteWrites := 0
	for i := range files {
		// Optionally Flatten Batches
		if m.cfg.Merging.FlattenBatches != nil {
			if file, err := files[i].FlattenBatches(); err != nil {
				el.Add(err)
			} else {
				files[i] = file
			}
		}
		// Write our file to the mergable directory
		if err := saveMergedFile(dir, files[i]); err != nil {
			el.Add(fmt.Errorf("problem writing merged file: %v", err))
		}
		// Perform the file upload if we are the shard leader
		if isLeader, err := m.consul.Acquire(shardKey); isLeader && err == nil {
			if err := f(agent, files[i]); err != nil {
				el.Add(fmt.Errorf("problem from callback: %v", err))
			} else {
				successfulRemoteWrites++
			}
		} else {
			logger.Logf("skipping file upload for shardKey=%s", shardKey)
		}
	}

	logger.Logf("wrote %d of %d files to remote agent", successfulRemoteWrites, len(files))

	if !el.Empty() {
		return nil, el
	}

	return newProcessedTransfers(matches), nil
}

func saveMergedFile(dir string, file *ach.File) error {
	var buf bytes.Buffer
	if err := ach.NewWriter(&buf).Write(file); err != nil {
		return fmt.Errorf("unable to buffer ACH file: %v", err)
	}
	filename := filepath.Join(dir, fmt.Sprintf("%s.ach", hash(buf.Bytes())))
	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

func hash(data []byte) string {
	ss := sha256.New()
	ss.Write(data)
	return hex.EncodeToString(ss.Sum(nil))
}
