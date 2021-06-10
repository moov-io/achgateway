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

package odfi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base"
	"github.com/moov-io/base/log"
)

// Cleanup deletes files on remote servers if enabled via config
func Cleanup(logger log.Logger, agent upload.Agent, dl *downloadedFiles) error {
	var el base.ErrorList

	if err := deleteFilesOnRemote(logger, agent, dl.dir, agent.InboundPath()); err != nil {
		el.Add(err)
	}
	if err := deleteFilesOnRemote(logger, agent, dl.dir, agent.ReconciliationPath()); err != nil {
		el.Add(err)
	}
	if err := deleteFilesOnRemote(logger, agent, dl.dir, agent.ReturnPath()); err != nil {
		el.Add(err)
	}
	if el.Empty() {
		return nil
	}
	return el
}

// CleanupEmptyFiles deletes empty ACH files if file is older than value in config
func CleanupEmptyFiles(logger log.Logger, agent upload.Agent, dl *downloadedFiles) error {
	var el base.ErrorList
	for _, path := range []string{agent.InboundPath(), agent.ReconciliationPath(), agent.ReturnPath()} {
		if _, err := os.Stat(filepath.Join(dl.dir, path)); err != nil {
			continue // skip if the directory doesn't exist
		}
		if err := deleteEmptyFiles(logger, agent, dl.dir, path); err != nil {
			el.Add(err)
		}
	}
	if el.Empty() {
		return nil
	}
	return el
}

// deleteFilesOnRemote deletes all files for a given directory
func deleteFilesOnRemote(logger log.Logger, agent upload.Agent, localDir, suffix string) error {
	baseDir := filepath.Join(localDir, suffix)
	infos, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("reading %s: %v", baseDir, err)
	}

	var el base.ErrorList
	for i := range infos {
		path := filepath.Join(suffix, filepath.Base(infos[i].Name()))
		if err := agent.Delete(path); err != nil {
			// Ignore the error if it's about deleting a remote file that's gone
			if os.IsNotExist(err) {
				continue
			}
			el.Add(err)
		} else {
			logger.Logf("cleanup: deleted remote file %s", path)
		}
	}

	if el.Empty() {
		return nil
	}
	return el
}

// deleteEmptyFiles deletes all empty files that are older than after (time.Duration)
func deleteEmptyFiles(logger log.Logger, agent upload.Agent, localDir, suffix string) error {
	baseDir := filepath.Join(localDir, suffix)
	infos, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return fmt.Errorf("reading %s: %v", baseDir, err)
	}

	var el base.ErrorList
	for i := range infos {
		path := filepath.Join(suffix, filepath.Base(infos[i].Name()))

		if infos[i].Size() != 0 {
			logger.Logf("file %s not deleted", path)
			continue
		}

		// Delete local file
		os.Remove(infos[i].Name())

		// Go ahead and delete the remote file
		if err := agent.Delete(path); err != nil {
			el.Add(err)
		}

		logger.Logf("deleted zero byte file %s", path)
	}

	if el.Empty() {
		return nil
	}
	return el
}
