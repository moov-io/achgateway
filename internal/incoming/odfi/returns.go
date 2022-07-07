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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	returnEntriesProcessed = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "return_entries_processed",
		Help: "Counter of return EntryDetail records processed",
	}, []string{"origin", "destination", "code"})
)

type returnEmitter struct {
	logger log.Logger
	svc    events.Emitter
	cfg    service.ODFIReturns
}

func ReturnEmitter(logger log.Logger, cfg service.ODFIReturns, svc events.Emitter) *returnEmitter {
	if !cfg.Enabled {
		return nil
	}
	return &returnEmitter{
		logger: logger,
		svc:    svc,
		cfg:    cfg,
	}
}

func (pc *returnEmitter) Type() string {
	return "return"
}

func isReturnFile(file File) bool {
	return len(file.ACHFile.ReturnEntries) >= 0
}

func (pc *returnEmitter) Handle(file File) error {
	if !isReturnFile(file) {
		return nil
	}

	// Ignore files if they don't contain the PathMatcher value
	if pc.cfg.PathMatcher != "" && !strings.Contains(strings.ToLower(file.Filepath), pc.cfg.PathMatcher) {
		return nil // skip the file
	}

	msg := models.ReturnFile{
		Filename: filepath.Base(file.Filepath),
		File:     file.ACHFile,
	}

	pc.logger.With(log.Fields{
		"origin":      log.String(file.ACHFile.Header.ImmediateOrigin),
		"destination": log.String(file.ACHFile.Header.ImmediateDestination),
	}).Log("odfi: processing return file")

	for i := range file.ACHFile.ReturnEntries {
		entries := file.ACHFile.ReturnEntries[i].GetEntries()
		msg.Returns = append(msg.Returns, models.Batch{
			Header:  file.ACHFile.ReturnEntries[i].GetHeader(),
			Entries: entries,
		})
		for j := range entries {
			if entries[j].Addenda99 == nil {
				continue
			}

			returnCode := entries[j].Addenda99.ReturnCodeField()
			returnEntriesProcessed.With(
				"origin", file.ACHFile.Header.ImmediateOrigin,
				"destination", file.ACHFile.Header.ImmediateDestination,
				"code", returnCode.Code,
			).Add(1)

			pc.logger.With(log.Fields{
				"origin":      log.String(file.ACHFile.Header.ImmediateOrigin),
				"destination": log.String(file.ACHFile.Header.ImmediateDestination),
			}).Log(fmt.Sprintf("odfi: return batch %d entry %d code %s", i, j, returnCode.Code))
		}
	}
	pc.sendEvent(msg)
	return nil
}

func (pc *returnEmitter) sendEvent(event interface{}) {
	if pc.svc != nil {
		err := pc.svc.Send(models.Event{Event: event})
		if err != nil {
			pc.logger.Logf("error sending return event: %v", err)
		}
	}
}
