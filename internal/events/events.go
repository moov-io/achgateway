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

package events

import (
	"bytes"
	"encoding/json"
	"reflect"
	"time"

	"github.com/moov-io/ach"
)

type Event struct {
	Event interface{} `json:"event"`
	Type  string      `json:"type"`
}

func (evt Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Event interface{} `json:"event"`
		Type  string      `json:"type"`
	}{
		Event: evt.Event,
		Type:  reflect.TypeOf(evt.Event).Name(),
	})
}

type CorrectionFile struct {
	File *ach.File `json:"file"`
}

type IncomingFile struct {
	File *ach.File `json:"file"`
}

type ReturnFile struct {
	File *ach.File `json:"file"`
}

type FileUploaded struct {
	FileID     string    `json:"fileID"`
	ShardKey   string    `json:"shardKey"`
	UploadedAt time.Time `json:"uploadedAt"`
}

func (f FileUploaded) Bytes() []byte {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(f)
	return buf.Bytes()
}
