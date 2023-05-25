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
	correctionCodesProcessed = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "correction_codes_processed",
		Help: "Counter of correction (COR/NOC) files processed",
	}, []string{"origin", "destination", "code"})
)

type correctionProcessor struct {
	svc events.Emitter
	cfg service.ODFICorrections
}

func CorrectionEmitter(cfg service.ODFICorrections, svc events.Emitter) *correctionProcessor {
	if !cfg.Enabled {
		return nil
	}
	return &correctionProcessor{
		svc: svc,
		cfg: cfg,
	}
}

func (pc *correctionProcessor) Type() string {
	return "correction"
}

func isCorrectionFile(file File) bool {
	return len(file.ACHFile.NotificationOfChange) >= 0
}

func (pc *correctionProcessor) Handle(logger log.Logger, file File) error {
	if !isCorrectionFile(file) {
		return nil
	}

	// Ignore files if they don't contain the PathMatcher value
	if pc.cfg.PathMatcher != "" && !strings.Contains(strings.ToLower(file.Filepath), pc.cfg.PathMatcher) {
		return nil // skip the file
	}

	msg := models.CorrectionFile{
		Filename: filepath.Base(file.Filepath),
		File:     file.ACHFile,
	}

	logger = logger.With(log.Fields{
		"origin":      log.String(file.ACHFile.Header.ImmediateOrigin),
		"destination": log.String(file.ACHFile.Header.ImmediateDestination),
	})
	logger.Logf("inbound: correction for %d batches", len(file.ACHFile.NotificationOfChange))

	for i := range file.ACHFile.NotificationOfChange {
		entries := file.ACHFile.NotificationOfChange[i].GetEntries()
		msg.Corrections = append(msg.Corrections, models.Batch{
			Header:  file.ACHFile.NotificationOfChange[i].GetHeader(),
			Entries: entries,
		})

		for j := range entries {
			if entries[j].Addenda98 == nil {
				continue
			}
			changeCode := entries[j].Addenda98.ChangeCodeField()
			correctionCodesProcessed.With(
				"origin", file.ACHFile.Header.ImmediateOrigin,
				"destination", file.ACHFile.Header.ImmediateDestination,
				"code", changeCode.Code,
			).Add(1)

			logger.With(log.Fields{
				"origin":      log.String(file.ACHFile.Header.ImmediateOrigin),
				"destination": log.String(file.ACHFile.Header.ImmediateDestination),
			}).Log(fmt.Sprintf("odfi: correction batch %d entry %d code %s", i, j, changeCode.Code))
		}
	}
	return pc.sendEvent(msg)
}

func (pc *correctionProcessor) sendEvent(event interface{}) error {
	if pc.svc != nil {
		err := pc.svc.Send(models.Event{Event: event})
		if err != nil {
			return fmt.Errorf("sending correction event: %w", err)
		}
	}
	return nil
}
