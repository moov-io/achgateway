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

package incoming

import (
	"errors"

	"github.com/moov-io/ach"
)

type ACHFile struct {
	FileID   string    `json:"id"`
	ShardKey string    `json:"shardKey"`
	File     *ach.File `json:"file"`
}

func (f ACHFile) Validate() error {
	if f.FileID == "" {
		return errors.New("missing fileID")
	}
	if f.ShardKey == "" {
		return errors.New("missing shardKey")
	}
	if f.File == nil {
		return errors.New("missing File")
	}
	return nil
}

type QueueACHFileResponse struct {
	FileID   string `json:"id"`
	ShardKey string `json:"shardKey"`
	Error    string `json:"error"`
}

type CancelACHFile struct {
	FileID   string `json:"id"`
	ShardKey string `json:"shardKey"`
}

type FileCancellationResponse struct {
	FileID     string `json:"id"`
	ShardKey   string `json:"shardKey"`
	Successful bool   `json:"successful"`
}
