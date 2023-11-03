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
	"context"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/upload"
)

type MockXferMerging struct {
	LatestFile   *incoming.ACHFile
	LatestCancel *incoming.CancelACHFile
	processed    *processedFiles

	Err error
}

func (merge *MockXferMerging) HandleXfer(_ context.Context, xfer incoming.ACHFile) error {
	merge.LatestFile = &xfer
	return merge.Err
}

func (merge *MockXferMerging) HandleCancel(_ context.Context, cancel incoming.CancelACHFile) error {
	merge.LatestCancel = &cancel
	return merge.Err
}

func (merge *MockXferMerging) WithEachMerged(_ context.Context, f func(context.Context, int, upload.Agent, *ach.File) error) (*processedFiles, error) {
	if merge.Err != nil {
		return nil, merge.Err
	}
	return merge.processed, nil
}
