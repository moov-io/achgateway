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

// Event is a wrapper for all events sent to or received from ACHGateway.
// It's used to determine the underlying event type.
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
	return ReadWithOpts(data, &ach.ValidateOpts{})
}

// ReadWithOpts will unmarshal an event with ACH Validation Options
func ReadWithOpts(data []byte, opts *ach.ValidateOpts) (*Event, error) {
	var eventType struct {
		Type  string          `json:"type"`
		Event json.RawMessage `json:"event"`
	}
	err := json.Unmarshal(data, &eventType)
	if err != nil {
		return nil, fmt.Errorf("reading type: %v", err)
	}

	event := &Event{
		Type: eventType.Type,
	}

	switch eventType.Type {
	case "CorrectionFile":
		var file CorrectionFile
		file.SetValidation(opts)
		event.Event = &file

	case "IncomingFile":
		var file IncomingFile
		file.SetValidation(opts)
		event.Event = &file

	case "PrenoteFile":
		var file PrenoteFile
		file.SetValidation(opts)
		event.Event = &file

	case "ReconciliationEntry":
		var entry ReconciliationEntry
		entry.SetValidation(opts)
		event.Event = &entry

	case "ReconciliationFile":
		var file ReconciliationFile
		file.SetValidation(opts)
		event.Event = &file

	case "ReturnFile":
		var file ReturnFile
		file.SetValidation(opts)
		event.Event = &file

	case "ACHFile", "QueueACHFile":
		var file QueueACHFile
		file.SetValidation(opts)
		event.Event = &file

	case "InvalidQueueFile":
		var file InvalidQueueFile
		file.SetValidation(opts)
		event.Event = &file

	case "CancelACHFile":
		var file CancelACHFile
		event.Event = &file

	case "FileCancellationResponse":
		var response FileCancellationResponse
		event.Event = &response

	case "FileUploaded":
		var file FileUploaded
		event.Event = &file
	}

	err = json.Unmarshal(eventType.Event, event.Event)
	if err != nil {
		err = fmt.Errorf("reading %s failed: %w", eventType.Type, err)
	}
	return event, err
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
//
// File.ID will be set to a hash of the Nacha contents.
//
// See the Event struct for wrapping steps.
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
//
// File.ID will be set to a hash of the Nacha contents.
//
// See the Event struct for wrapping steps.
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
//
// File.ID will be set to a hash of the Nacha contents.
//
// See the Event struct for wrapping steps.
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

// ReconciliationEntry is an ACH entry that was initiated with the ODFI.
//
// See the Event struct for wrapping steps.
type ReconciliationEntry struct {
	Filename string           `json:"filename"`
	Header   *ach.BatchHeader `json:"batchHeader"`
	Entry    *ach.EntryDetail `json:"entry"`
}

func (evt *ReconciliationEntry) SetValidation(opts *ach.ValidateOpts) {
	if evt.Header == nil {
		evt.Header = ach.NewBatchHeader()
	}
	evt.Header.SetValidation(opts)

	if evt.Entry == nil {
		evt.Entry = ach.NewEntryDetail()
	}
	evt.Entry.SetValidation(opts)
}

// ReconciliationFile is a file whose entries match entries initiated with the ODFI.
//
// File.ID will be set to a hash of the Nacha contents.
//
// See the Event struct for wrapping steps.
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
//
// File.ID will be set to a hash of the Nacha contents.
//
// See the Event struct for wrapping steps.
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
//
// See the Event struct for wrapping steps.
type QueueACHFile incoming.ACHFile

func (evt *QueueACHFile) SetValidation(opts *ach.ValidateOpts) {
	if evt.File == nil {
		evt.File = ach.NewFile()
	}
	evt.File.SetValidation(opts)
}

// QueueACHFileResponse is a response to the QueueACHFile event signaling if the file was successfully enqueued.
type QueueACHFileResponse incoming.QueueACHFileResponse

// InvalidQueueFile is an event that achgateway produces when a QueueACHFile could not be processed.
// This event is typically produced when the ACH file is invalid.
type InvalidQueueFile struct {
	File     QueueACHFile `json:"file"`
	Error    string       `json:"error"`
	Hostname string       `json:"hostname"`
}

func (evt *InvalidQueueFile) SetValidation(opts *ach.ValidateOpts) {
	if opts == nil {
		opts = &ach.ValidateOpts{}
	}
	opts.SkipAll = true
	evt.File.SetValidation(opts)
}

// CancelACHFile is an event that achgateway receives to cancel uploading a file to the ODFI.
//
// See the Event struct for wrapping steps.
type CancelACHFile incoming.CancelACHFile

// FileCancellationResponse is a response to the CancelACHFile event signaling if the cancellation was successful.
type FileCancellationResponse incoming.FileCancellationResponse

// FileUploaded is an event sent after a queued file has been uploaded to the ODFI.
// The entries and batches may have been merged into a larger file to optimize on cost,
// network performance, or other configuration.
type FileUploaded struct {
	FileID     string    `json:"fileID"`
	ShardKey   string    `json:"shardKey"`
	Filename   string    `json:"filename"`
	UploadedAt time.Time `json:"uploadedAt"`
}
