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

package odfi

import (
	"crypto/sha1" //nolint:gosec
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/alerting"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base"
	"github.com/moov-io/base/log"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	processingErrors = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "inbound_processing_errors",
		Help: "Counter of errors encountered when downloading or processing inbound files",
	}, []string{"processor"})
)

type File struct {
	Filepath string
	ACHFile  *ach.File
}

type FileProcessor interface {
	Type() string

	// Handle processes an ACH file with whatever logic is implemented
	Handle(logger log.Logger, file File) error
}

type Processors []FileProcessor

func SetupProcessors(pcs ...FileProcessor) Processors {
	var out Processors
	for i := range pcs {
		v := reflect.ValueOf(pcs[i])
		if !v.IsNil() {
			out = append(out, pcs[i])
		}
	}
	return out
}

func (pcs Processors) HandleAll(logger log.Logger, file File) error {
	var el base.ErrorList
	for i := range pcs {
		proc := pcs[i]
		if err := proc.Handle(logger, file); err != nil {
			processingErrors.With("processor", fmt.Sprintf("%T", proc)).Add(1)

			el.Add(fmt.Errorf("%s: %v", proc.Type(), err))
		}
	}
	if el.Empty() {
		return nil
	}
	return el
}

func ProcessFiles(logger log.Logger, dl *downloadedFiles, alerters alerting.Alerters, auditSaver *AuditSaver, validation ach.ValidateOpts, fileProcessors Processors, agent upload.Agent) error {
	var el base.ErrorList

	for _, processingPath := range []string{agent.InboundPath(), agent.ReconciliationPath(), agent.ReturnPath()} {
		where := filepath.Join(dl.dir, processingPath)
		if err := processDir(logger, where, alerters, auditSaver, validation, fileProcessors); err != nil {
			el.Add(fmt.Errorf("processDir %s: %v", where, err))
		}
	}

	if el.Empty() {
		return nil
	}
	return el
}

func processDir(logger log.Logger, dir string, alerters alerting.Alerters, auditSaver *AuditSaver, validation ach.ValidateOpts, fileProcessors Processors) error {
	infos, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %v", dir, err)
	}

	var el base.ErrorList
	for _, info := range infos {
		where := filepath.Join(dir, info.Name())
		logger = logger.With(log.Fields{
			"filename": log.String(where),
		})
		if err := processFile(logger, where, alerters, auditSaver, validation, fileProcessors); err != nil {
			el.Add(err)
		}
	}

	if el.Empty() {
		return nil
	}
	return el
}

func processFile(logger log.Logger, path string, alerters alerting.Alerters, auditSaver *AuditSaver, validation ach.ValidateOpts, fileProcessors Processors) error {
	fileHandle, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("problem opening %s: %v", path, err)
	}
	//bs = bytes.TrimSpace(bs)

	hashReader := NewHashFilter(fileHandle)

	reader := ach.NewReader(hashReader)
	// Enable some default ACH ValidateOpts
	validation.AllowMissingFileControl = true
	validation.AllowMissingFileHeader = true
	validation.AllowUnorderedBatchNumbers = true
	reader.SetValidation(&validation)

	file, err := reader.Read()
	if err != nil {
		// Some return files don't contain FileHeader info, but can be processed as there
		// are batches with entries. Let's continue to process those, but skip other errors.
		if !base.Has(err, ach.ErrFileHeader) {
			return fmt.Errorf("problem parsing %s: %v", path, err)
		}
	}
	file.ID = fmt.Sprintf("%x", hashReader.Sum())
	populateHashes(&file)

	dir, filename := filepath.Split(path)
	dir = filepath.Base(dir)

	// Persist the file if needed
	if auditSaver != nil {
		path := fmt.Sprintf("odfi/%s/%s/%s/%s", auditSaver.hostname, dir, time.Now().Format("2006-01-02"), filename)
		_, err = fileHandle.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("audittrail %s error: %v", path, err)
		}
		err = auditSaver.save(path, fileHandle)
		if err != nil {
			return fmt.Errorf("audittrail %s error: %v", path, err)
		}
	}

	// Pass the file off to our handler
	err = fileProcessors.HandleAll(logger, File{
		Filepath: path,
		ACHFile:  &file,
	})
	if err != nil {
		alertErr := alerters.AlertError(err)
		if alertErr != nil {
			return fmt.Errorf("problem alerting on error: %w", alertErr)
		}
		return fmt.Errorf("processing %s error: %v", path, err)
	}

	return nil
}

func populateHashes(file *ach.File) {
	for i := range file.Batches {
		entries := file.Batches[i].GetEntries()
		for j := range entries {
			entries[j].ID = hash([]byte(entries[j].String()))
		}
	}
}

func hash(data []byte) string {
	return fmt.Sprintf("%x", sha1.Sum(data)) //nolint:gosec
}
