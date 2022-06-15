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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"time"

	"github.com/moov-io/ach"
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

func ProcessFiles(dl *downloadedFiles, auditSaver *AuditSaver, fileProcessors Processors) error {
	var el base.ErrorList
	dirs, err := ioutil.ReadDir(dl.dir)
	if err != nil {
		return fmt.Errorf("reading %s: %v", dl.dir, err)
	}
	for i := range dirs {
		where := filepath.Join(dl.dir, dirs[i].Name())

		if dirs[i].Mode().IsDir() {
			err = processDir(where, auditSaver, fileProcessors)
			if err != nil {
				el.Add(fmt.Errorf("processDir %s: %v", dirs[i], err))
				continue
			}
		}
		if dirs[i].Mode().IsRegular() {
			err = processFile(where, auditSaver, fileProcessors)
			if err != nil {
				el.Add(fmt.Errorf("processfile - %s: %v", dirs[i], err))
				continue
			}
		}
	}
	if el.Empty() {
		return nil
	}
	return el
}

func processDir(dir string, auditSaver *AuditSaver, fileProcessors Processors) error {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %v", dir, err)
	}

	var el base.ErrorList
	for i := range infos {
		where := filepath.Join(dir, infos[i].Name())

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
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("problem opening %s: %v", path, err)
	}

	// Parse the ACH file
	file, err := ach.NewReader(bytes.NewReader(bs)).Read()
	if err != nil {
		// Some return files don't contain FileHeader info, but can be processed as there
		// are batches with entries. Let's continue to process those, but skip other errors.
		if !base.Has(err, ach.ErrFileHeader) {
			return fmt.Errorf("problem parsing %s: %v", path, err)
		}
	}

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
