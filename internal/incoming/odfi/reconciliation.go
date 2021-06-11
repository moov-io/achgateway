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
	"errors"
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
	creditReconciliationFilesProcessed = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "credit_reconciliation_files_processed",
		Help: "Counter of Credit Reconciliation files encountered",
	}, []string{"origin", "destination"})
)

type creditReconciliation struct {
	logger log.Logger
	svc    events.Emitter
	cfg    service.ODFIReconciliation
}

func CreditReconciliationEmitter(logger log.Logger, cfg service.ODFIReconciliation, svc events.Emitter) *creditReconciliation {
	if !cfg.Enabled {
		return nil
	}
	return &creditReconciliation{
		logger: logger,
		svc:    svc,
		cfg:    cfg,
	}
}

func (pc *creditReconciliation) Type() string {
	return "CreditReconciliation"
}

func (pc *creditReconciliation) Handle(file File) error {
	if file.ACHFile == nil {
		return errors.New("nil ach.File")
	}

	// For now we are inspecting the filepath to see if it came from our
	// configured reconciliation path. That's the best source of information
	// for when we should treat the file as a recon file.
	//
	// Example: /reconciliation/fileMoovTester_TRANACTIONSFAKE.TXT
	if !strings.Contains(strings.ToLower(file.Filepath), pc.cfg.PatchMatcher) {
		return nil // skip the file
	}

	// Record that we've encountered this ACH file
	creditReconciliationFilesProcessed.With(
		"origin", file.ACHFile.Header.ImmediateOrigin,
		"destination", file.ACHFile.Header.ImmediateDestination,
	).Add(1)
	pc.logger.With(log.Fields{
		"filepath": log.String(file.Filepath),
	}).Log("odfi: processing reconciliation file")

	var recons []models.Batch

	// Attempt to match each Transfer
	for i := range file.ACHFile.Batches {
		batch := models.Batch{
			Header: file.ACHFile.Batches[i].GetHeader(),
		}
		entries := file.ACHFile.Batches[i].GetEntries()
		for j := range entries {
			if !isCreditEntry(entries[j]) {
				pc.recordSkippingEntry(entries[j])
				continue
			}

			pc.logger.With(log.Fields{
				"traceNumber": log.String(entries[j].TraceNumber),
			}).Log("odfi: received reconciliation entry")

			// Save off event information
			batch.Entries = append(batch.Entries, entries[j])
		}
		if len(batch.Entries) > 0 {
			recons = append(recons, batch)
		}
	}

	if len(recons) > 0 {
		pc.svc.Send(models.Event{
			Event: models.ReconciliationFile{
				Filename:        filepath.Base(file.Filepath),
				File:            file.ACHFile,
				Reconciliations: recons,
			},
		})
	}

	return nil
}

func isCreditEntry(ed *ach.EntryDetail) bool {
	switch ed.TransactionCode {
	case ach.CheckingCredit, ach.SavingsCredit:
		return true
	}
	return false
}

func (pc *creditReconciliation) recordSkippingEntry(ed *ach.EntryDetail) {
	var changeCode string
	if ed.Addenda98 != nil {
		changeCode = ed.Addenda98.ChangeCode
	}
	var returnCode string
	if ed.Addenda99 != nil {
		returnCode = ed.Addenda99.ReturnCode
	}
	pc.logger.With(log.Fields{
		"changeCode":      log.String(changeCode),
		"returnCode":      log.String(returnCode),
		"traceNumber":     log.String(ed.TraceNumber),
		"transactionCode": log.Int(ed.TransactionCode),
	}).Log("skipping entry")
}
