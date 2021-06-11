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

package models

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

func (evt Event) Bytes() []byte {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(evt)
	return buf.Bytes()
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

func ReadEvent(data []byte, evt interface{}) error {
	return json.Unmarshal(data, &Event{
		Event: evt,
	})
}

type Batch struct {
	Header  *ach.BatchHeader   `json:"batchHeader"`
	Entries []*ach.EntryDetail `json:"entryDetails"`
}

type CorrectionFile struct {
	Filename    string    `json:"filename"`
	File        *ach.File `json:"file"`
	Corrections []Batch   `json:"corrections"`
}

type IncomingFile struct {
	Filename string    `json:"filename"`
	File     *ach.File `json:"file"`
}

type PrenoteFile struct {
	Filename string    `json:"filename"`
	File     *ach.File `json:"file"`
	Batches  []Batch   `json:"batches"`
}

type ReconciliationFile struct {
	Filename        string    `json:"filename"`
	File            *ach.File `json:"file"`
	Reconciliations []Batch   `json:"returns"`
}

type ReturnFile struct {
	Filename string    `json:"filename"`
	File     *ach.File `json:"file"`
	Returns  []Batch   `json:"returns"`
}

type FileUploaded struct {
	FileID     string    `json:"fileID"`
	ShardKey   string    `json:"shardKey"`
	Filename   string    `json:"filename"`
	UploadedAt time.Time `json:"uploadedAt"`
}
