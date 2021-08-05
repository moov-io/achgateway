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
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestCleanupErr(t *testing.T) {
	agent := &upload.MockAgent{
		Err: errors.New("bad error"),
	}

	dir, _ := ioutil.TempDir("", "clenaup-testing")
	dl := &downloadedFiles{dir: dir}
	defer dl.deleteFiles()

	// write a test file to attempt deletion
	path := filepath.Join(dl.dir, agent.InboundPath())
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(path, "file.ach"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	// test out cleanup func
	if err := Cleanup(log.NewNopLogger(), agent, dl); err == nil {
		t.Error("expected error")
	}

	if agent.DeletedFile != "inbound/file.ach" {
		t.Errorf("unexpected deleted file: %s", agent.DeletedFile)
	}
}

func Test_CleanupEmptyFiles_InboundPath_Success(t *testing.T) {
	agent := &upload.MockAgent{}

	dir, _ := ioutil.TempDir("", "clenaup-testing")
	dl := &downloadedFiles{dir: dir}
	defer dl.deleteFiles()

	// write a test file to attempt deletion
	path := filepath.Join(dl.dir, agent.InboundPath())
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(path, "empty_file.ach"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	err := CleanupEmptyFiles(log.NewNopLogger(), agent, dl)
	require.NoError(t, err)

	if agent.DeletedFile != "inbound/empty_file.ach" {
		t.Errorf("unexpected deleted file: %s", agent.DeletedFile)
	}
}

func Test_CleanupEmptyFiles_ReconciliationPath_Success(t *testing.T) {
	agent := &upload.MockAgent{}

	dir, _ := ioutil.TempDir("", "clenaup-testing")
	dl := &downloadedFiles{dir: dir}
	defer dl.deleteFiles()

	// write a test file to attempt deletion
	path := filepath.Join(dl.dir, agent.ReconciliationPath())
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(path, "empty_file.ach"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	err := CleanupEmptyFiles(log.NewNopLogger(), agent, dl)
	require.NoError(t, err)

	if agent.DeletedFile != "reconciliation/empty_file.ach" {
		t.Errorf("unexpected deleted file: %s", agent.DeletedFile)
	}
}

func Test_CleanupEmptyFiles_ReturnPath_Success(t *testing.T) {
	agent := &upload.MockAgent{}

	dir, _ := ioutil.TempDir("", "clenaup-testing")
	dl := &downloadedFiles{dir: dir}
	defer dl.deleteFiles()

	// write a test file to attempt deletion
	path := filepath.Join(dl.dir, agent.ReturnPath())
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(path, "empty_file.ach"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	err := CleanupEmptyFiles(log.NewNopLogger(), agent, dl)
	require.NoError(t, err)

	if agent.DeletedFile != "return/empty_file.ach" {
		t.Errorf("unexpected deleted file: %s", agent.DeletedFile)
	}
}

func Test_CleanupEmptyFiles_PopulatedFile(t *testing.T) {
	agent := &upload.MockAgent{}

	dir, _ := ioutil.TempDir("", "clenaup-testing")
	dl := &downloadedFiles{dir: dir}
	defer dl.deleteFiles()

	// write a test file to attempt deletion
	path := filepath.Join(dl.dir, agent.InboundPath())
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(filepath.Join(path, "file.ach"), []byte("sameple data"), 0600); err != nil {
		t.Fatal(err)
	}

	err := CleanupEmptyFiles(log.NewNopLogger(), agent, dl)
	require.NoError(t, err)

	if agent.DeletedFile != "" {
		t.Errorf("expected no deleted files, but got %q", agent.DeletedFile)
	}
}
