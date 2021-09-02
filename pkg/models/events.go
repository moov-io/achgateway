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
	"github.com/moov-io/achgateway/internal/incoming"
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

// CorrectionFile is an event for when an Addenda98 record is found within a file
// from the ODFI. This is also called a "Notification of Change" (NOC).
type CorrectionFile struct {
	Filename    string    `json:"filename"`
	File        *ach.File `json:"file"`
	Corrections []Batch   `json:"corrections"`
}

// IncomingFile is an event for when an ODFI receives an ACH file from another FI
// signifying entries to process (e.g. another FI is debiting your account).
type IncomingFile struct {
	Filename string    `json:"filename"`
	File     *ach.File `json:"file"`
}

// PrenoteFile is an event for when an ODFI receives a "pre-notification" ACH file.
// This type of file is used to validate accounts exist and are usable for ACH.
type PrenoteFile struct {
	Filename string    `json:"filename"`
	File     *ach.File `json:"file"`
	Batches  []Batch   `json:"batches"`
}

// ReconciliationFile is a file whose entries match entries initiated with the ODFI.
type ReconciliationFile struct {
	Filename        string    `json:"filename"`
	File            *ach.File `json:"file"`
	Reconciliations []Batch   `json:"returns"`
}

func (evt *ReconciliationFile) SetValidation(opts *ach.ValidateOpts) {
	if evt.File == nil {
		evt.File = ach.NewFile()
	}
	evt.File.SetValidation(opts)
}

// ReturnFile is an event for when an Addenda99 record is found within a file
// from the ODFI. This is also called a "return".
type ReturnFile struct {
	Filename string    `json:"filename"`
	File     *ach.File `json:"file"`
	Returns  []Batch   `json:"returns"`
}

// QueueACHFile is an event that achgateway receives to enqueue an ACH file for upload to the
// ODFI at a later cutoff time.
type QueueACHFile incoming.ACHFile

// CancelACHFile is an event that achgateway receives to cancel uploading a file to the ODFI.
type CancelACHFile incoming.ACHFile

// FileUploaded is an event sent after a queued file has been uploaded to the ODFI.
// The entries and batches may have been merged into a larger file to optimize on cost,
// network performance, or other configuration.
type FileUploaded struct {
	FileID     string    `json:"fileID"`
	ShardKey   string    `json:"shardKey"`
	Filename   string    `json:"filename"`
	UploadedAt time.Time `json:"uploadedAt"`
}
