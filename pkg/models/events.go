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
	"fmt"
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

// Read will unmarshal an event and return the wrapper for it.
func Read(data []byte) (*Event, error) {
	var eventType struct {
		Type string `json:"type"`
	}
	err := json.Unmarshal(data, &eventType)
	if err != nil {
		return nil, fmt.Errorf("reading type: %v", err)
	}

	var evt interface{}
	switch eventType.Type {
	case "CorrectionFile":
		evt = &CorrectionFile{}
	case "IncomingFile":
		evt = &IncomingFile{}
	case "PrenoteFile":
		evt = &PrenoteFile{}
	case "ReconciliationFile":
		evt = &ReconciliationFile{}
	case "ReturnFile":
		evt = &ReturnFile{}
	case "ACHFile", "QueueACHFile":
		evt = &QueueACHFile{}
	case "CancelACHFile":
		evt = &CancelACHFile{}
	}

	err = ReadEvent(data, evt)
	if err != nil {
		return nil, fmt.Errorf("reading event: %v", err)
	}
	return &Event{
		Event: evt,
		Type:  eventType.Type,
	}, nil
}

// ReadEvent will unmarshal the event, but assumes the event type is known by the caller.
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

func (evt *CorrectionFile) SetValidation(opts *ach.ValidateOpts) {
	if evt.File == nil {
		evt.File = ach.NewFile()
	}
	evt.File.SetValidation(opts)
}

// IncomingFile is an event for when an ODFI receives an ACH file from another FI
// signifying entries to process (e.g. another FI is debiting your account).
type IncomingFile struct {
	Filename string    `json:"filename"`
	File     *ach.File `json:"file"`
}

func (evt *IncomingFile) SetValidation(opts *ach.ValidateOpts) {
	if evt.File == nil {
		evt.File = ach.NewFile()
	}
	evt.File.SetValidation(opts)
}

// PrenoteFile is an event for when an ODFI receives a "pre-notification" ACH file.
// This type of file is used to validate accounts exist and are usable for ACH.
type PrenoteFile struct {
	Filename string    `json:"filename"`
	File     *ach.File `json:"file"`
	Batches  []Batch   `json:"batches"`
}

func (evt *PrenoteFile) SetValidation(opts *ach.ValidateOpts) {
	if evt.File == nil {
		evt.File = ach.NewFile()
	}
	evt.File.SetValidation(opts)
}

// ReconciliationFile is a file whose entries match entries initiated with the ODFI.
type ReconciliationFile struct {
	Filename        string    `json:"filename"`
	File            *ach.File `json:"file"`
	Reconciliations []Batch   `json:"reconciliations"`
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

func (evt *ReturnFile) SetValidation(opts *ach.ValidateOpts) {
	if evt.File == nil {
		evt.File = ach.NewFile()
	}
	evt.File.SetValidation(opts)
}

// QueueACHFile is an event that achgateway receives to enqueue an ACH file for upload to the
// ODFI at a later cutoff time.
type QueueACHFile incoming.ACHFile

func (evt *QueueACHFile) SetValidation(opts *ach.ValidateOpts) {
	if evt.File == nil {
		evt.File = ach.NewFile()
	}
	evt.File.SetValidation(opts)
}

// CancelACHFile is an event that achgateway receives to cancel uploading a file to the ODFI.
type CancelACHFile incoming.CancelACHFile

// FileUploaded is an event sent after a queued file has been uploaded to the ODFI.
// The entries and batches may have been merged into a larger file to optimize on cost,
// network performance, or other configuration.
type FileUploaded struct {
	FileID     string    `json:"fileID"`
	ShardKey   string    `json:"shardKey"`
	Filename   string    `json:"filename"`
	UploadedAt time.Time `json:"uploadedAt"`
}
