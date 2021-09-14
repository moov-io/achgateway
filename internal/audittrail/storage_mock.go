// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package audittrail

import (
	"io"
	"io/ioutil"
	"strings"
)

type MockStorage struct {
	Err          error
	FileContents string
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

func (s *MockStorage) SaveFile(_ string, _ []byte) error {
	if s.Err != nil {
		uploadFilesErrors.With("type", "mock", "id", "mock").Add(1)
	} else {
		uploadedFilesCounter.With("type", "mock", "id", "mock").Add(1)
	}
	return s.Err
}

func (s *MockStorage) GetFile(_ string) (io.ReadCloser, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return ioutil.NopCloser(strings.NewReader(s.FileContents)), nil
}
