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
	"bytes"
	"crypto/sha1" //nolint:gosec
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base"

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
	Handle(file File) error
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

func (pcs Processors) HandleAll(file File) error {
	var el base.ErrorList
	for i := range pcs {
		proc := pcs[i]
		if err := proc.Handle(file); err != nil {
			processingErrors.With("processor", fmt.Sprintf("%T", proc)).Add(1)

			el.Add(fmt.Errorf("%s: %v", proc.Type(), err))
		}
	}
	if el.Empty() {
		return nil
	}
	return el
}

func ProcessFiles(dl *downloadedFiles, auditSaver *AuditSaver, fileProcessors Processors, agent upload.Agent) error {
	var el base.ErrorList

	for _, processingPath := range []string{agent.InboundPath(), agent.ReconciliationPath(), agent.ReturnPath()} {
		where := filepath.Join(dl.dir, processingPath)
		if err := processDir(where, auditSaver, fileProcessors); err != nil {
			el.Add(fmt.Errorf("processDir %s: %v", where, err))
		}
	}

	if el.Empty() {
		return nil
	}
	return el
}

func processDir(dir string, auditSaver *AuditSaver, fileProcessors Processors) error {
	infos, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %v", dir, err)
	}

	var el base.ErrorList
	for _, info := range infos {
		where := filepath.Join(dir, info.Name())

		if err := processFile(where, auditSaver, fileProcessors); err != nil {
			el.Add(err)
		}
	}

	if el.Empty() {
		return nil
	}
	return el
}

func processFile(path string, auditSaver *AuditSaver, fileProcessors Processors) error {
	bs, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("problem opening %s: %v", path, err)
	}
	bs = bytes.TrimSpace(bs)

	reader := ach.NewReader(bytes.NewReader(bs))
	reader.SetValidation(&ach.ValidateOpts{
		AllowMissingFileControl:    true,
		AllowMissingFileHeader:     true,
		AllowUnorderedBatchNumbers: true,
	})

	file, err := reader.Read()
	if err != nil {
		// Some return files don't contain FileHeader info, but can be processed as there
		// are batches with entries. Let's continue to process those, but skip other errors.
		if !base.Has(err, ach.ErrFileHeader) {
			return fmt.Errorf("problem parsing %s: %v", path, err)
		}
	}
	file.ID = hash(bs)
	populateHashes(&file)

	dir, filename := filepath.Split(path)
	dir = filepath.Base(dir)

	// Persist the file if needed
	if auditSaver != nil {
		path := fmt.Sprintf("odfi/%s/%s/%s/%s", auditSaver.hostname, dir, time.Now().Format("2006-01-02"), filename)
		err = auditSaver.save(path, bs)
		if err != nil {
			return fmt.Errorf("audittrail %s error: %v", path, err)
		}
	}

	// Pass the file off to our handler
	err = fileProcessors.HandleAll(File{
		Filepath: path,
		ACHFile:  &file,
	})
	if err != nil {
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
