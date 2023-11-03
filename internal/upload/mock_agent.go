// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"bytes"
	"context"
	"io"
	"sync"
)

type MockAgent struct {
	InboundFilepaths        []string
	ReconciliationFilepaths []string
	ReturnFilepaths         []string

	UploadedFile *File  // non-nil on file upload
	DeletedFile  string // filepath of last deleted file
	ReadableFile *File

	mu sync.RWMutex // protects all fields

	Err error
}

func (a *MockAgent) ID() string {
	return "mock-agent"
}

func (a *MockAgent) GetInboundFiles(_ context.Context) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.InboundFilepaths, nil
}

func (a *MockAgent) GetReconciliationFiles(_ context.Context) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.ReconciliationFilepaths, nil
}

func (a *MockAgent) GetReturnFiles(_ context.Context) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.ReturnFilepaths, nil
}

func (a *MockAgent) UploadFile(_ context.Context, f File) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// read f.contents before callers close the underlying os.Open file descriptor
	bs, _ := io.ReadAll(f.Contents)
	a.UploadedFile = &f
	a.UploadedFile.Contents = io.NopCloser(bytes.NewReader(bs))
	return nil
}

func (a *MockAgent) Delete(_ context.Context, path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.DeletedFile = path
	return nil
}

func (a *MockAgent) ReadFile(_ context.Context, path string) (*File, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.ReadableFile, nil
}

func (a *MockAgent) InboundPath() string {
	return "inbound/"
}

func (a *MockAgent) OutboundPath() string {
	return "outbound/"
}

func (a *MockAgent) ReconciliationPath() string {
	return "reconciliation/"
}

func (a *MockAgent) ReturnPath() string {
	return "return/"
}

func (a *MockAgent) Hostname() string {
	return "hostname"
}

func (a *MockAgent) Ping() error {
	return a.Err
}

func (a *MockAgent) Close() error {
	return nil
}
