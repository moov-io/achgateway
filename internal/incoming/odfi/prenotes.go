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

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	prenoteEntriesProcessed = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "prenote_entries_processed",
		Help: "Counter of prenote EntryDetail records processed",
	}, []string{"origin", "destination", "transactionCode"})
)

type prenoteEmitter struct {
	logger log.Logger
	svc    events.Emitter
	cfg    service.ODFIPrenotes
}

func PrenoteEmitter(logger log.Logger, cfg service.ODFIPrenotes, svc events.Emitter) *prenoteEmitter {
	if !cfg.Enabled {
		return nil
	}
	return &prenoteEmitter{
		logger: logger,
		svc:    svc,
		cfg:    cfg,
	}
}

func (pc *prenoteEmitter) Type() string {
	return "prenote"
}

func (pc *prenoteEmitter) Handle(file File) error {
	// Ignore files if they don't contain the PathMatcher value
	if pc.cfg.PathMatcher != "" && !strings.Contains(strings.ToLower(file.Filepath), pc.cfg.PathMatcher) {
		return nil // skip the file
	}

	var batches []models.Batch

	for i := range file.ACHFile.Batches {
		batch := models.Batch{
			Header: file.ACHFile.Batches[i].GetHeader(),
		}
		entries := file.ACHFile.Batches[i].GetEntries()
		for j := range entries {
			if ok, _ := isPrenoteEntry(entries[j]); !ok {
				continue
			} else {
				batch.Entries = append(batch.Entries, entries[j])
			}

			pc.logger.With(log.Fields{
				"origin":      log.String(file.ACHFile.Header.ImmediateOrigin),
				"destination": log.String(file.ACHFile.Header.ImmediateDestination),
			}).Log(fmt.Sprintf("odfi: pre-notification traceNumber=%s", entries[j].TraceNumber))

			prenoteEntriesProcessed.With(
				"origin", file.ACHFile.Header.ImmediateOrigin,
				"destination", file.ACHFile.Header.ImmediateDestination,
				"transactionCode", fmt.Sprintf("%d", entries[j].TransactionCode),
			).Add(1)
		}
	}
	if len(batches) > 0 {
		pc.sendEvent(models.PrenoteFile{
			Filename: filepath.Base(file.Filepath),
			File:     file.ACHFile,
			Batches:  batches,
		})
	}
	return nil
}

func (pc *prenoteEmitter) sendEvent(event interface{}) {
	if pc.svc != nil {
		err := pc.svc.Send(models.Event{Event: event})
		if err != nil {
			pc.logger.Logf("error sending pre-note event: %v", err)
		}
	}
}

func isPrenoteFile(file File) bool {
	for i := range file.ACHFile.Batches {
		entries := file.ACHFile.Batches[i].GetEntries()
		for j := range entries {
			isPrenote, _ := isPrenoteEntry(entries[j])
			if isPrenote {
				return true
			}
		}
	}
	return false
}

// isPrenoteEntry checks if a given EntryDetail matches the pre-notification
// criteria. Per NACHA rules that means a zero amount and prenote transaction code.
func isPrenoteEntry(entry *ach.EntryDetail) (bool, error) {
	switch entry.TransactionCode {
	case
		ach.CheckingPrenoteCredit, ach.CheckingPrenoteDebit,
		ach.SavingsPrenoteCredit, ach.SavingsPrenoteDebit,
		ach.GLPrenoteCredit, ach.GLPrenoteDebit, ach.LoanPrenoteCredit:
		if entry.Amount == 0 {
			return true, nil // valid prenotification entry
		} else {
			return true, fmt.Errorf("non-zero prenotification amount=%d", entry.Amount)
		}
	}
	return false, nil
}
