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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestDownloader__deleteFiles(t *testing.T) {
	factory := &downloaderImpl{
		logger:  log.NewTestLogger(),
		baseDir: t.TempDir(),
	}

	agent := &upload.MockAgent{}
	dl, err := factory.setup(agent)
	require.NoError(t, err)

	// write a file and expect it to be deleted
	path := filepath.Join(dl.dir, agent.InboundPath(), "foo.ach")
	if err := os.WriteFile(path, []byte("testing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := dl.deleteFiles(); err != nil {
		t.Fatal(err)
	}

	// read files
	fds, err := os.ReadDir(dl.dir)
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(fds) != 0 {
		t.Errorf("%d unexpected files", len(fds))
	}
}

func TestDownloader__deleteEmptyDirs(t *testing.T) {
	factory := &downloaderImpl{
		logger:  log.NewTestLogger(),
		baseDir: t.TempDir(),
	}

	agent := &upload.MockAgent{}
	dl, err := factory.setup(agent)
	require.NoError(t, err)

	// write a file and expect it to be deleted
	path := filepath.Join(dl.dir, agent.InboundPath(), "foo.ach")
	if err := os.WriteFile(path, []byte("testing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := dl.deleteEmptyDirs(context.Background(), agent); err != nil {
		t.Fatal(err)
	}

	// read files
	fds, err := os.ReadDir(dl.dir)
	require.NoError(t, err)
	if len(fds) != 1 {
		t.Fatalf("%d unexpected files", len(fds))
	}
	if n := fds[0].Name(); n != "inbound" {
		t.Errorf("unexpected %v", n)
	}
	// Check the file still exists
	if _, err := os.Stat(path); err != nil {
		t.Error(err)
	}
}
