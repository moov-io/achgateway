// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package audittrail

import (
	"bytes"
	"io"
)

type MockStorage struct {
	Err error

	SavedFilepath string
	SavedContents []byte
}

func newMockStorage() *MockStorage {
	// default values for metrics
	uploadFilesErrors.With("type", "mock", "id", "mock").Add(0)
	uploadedFilesCounter.With("type", "mock", "id", "mock").Add(0)

	return &MockStorage{}
}

func (s *MockStorage) Close() error {
	return s.Err
}

func (s *MockStorage) SaveFile(path string, data []byte) error {
	if s.Err != nil {
		uploadFilesErrors.With("type", "mock", "id", "mock").Add(1)
	} else {
		uploadedFilesCounter.With("type", "mock", "id", "mock").Add(1)

		s.SavedFilepath = path
		s.SavedContents = data
	}
	return s.Err
}

func (s *MockStorage) GetFile(_ string) (io.ReadCloser, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return io.NopCloser(bytes.NewReader(s.SavedContents)), nil
}
