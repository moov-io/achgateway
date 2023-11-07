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
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/telemetry"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

var (
	creditReconciliationFilesProcessed = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "credit_reconciliation_files_processed",
		Help: "Counter of Credit Reconciliation files encountered",
	}, []string{"origin", "destination"})
)

type creditReconciliation struct {
	svc events.Emitter
	cfg service.ODFIReconciliation
}

func CreditReconciliationEmitter(cfg service.ODFIReconciliation, svc events.Emitter) *creditReconciliation {
	if !cfg.Enabled {
		return nil
	}
	return &creditReconciliation{
		svc: svc,
		cfg: cfg,
	}
}

func (pc *creditReconciliation) Type() string {
	return "CreditReconciliation"
}

func isReconciliationFile(cfg service.ODFIReconciliation, file File) bool {
	if !cfg.Enabled {
		return false
	}
	return cfg.PathMatcher != "" && strings.Contains(strings.ToLower(file.Filepath), cfg.PathMatcher)
}

func (pc *creditReconciliation) Handle(ctx context.Context, logger log.Logger, file File) error {
	if file.ACHFile == nil {
		return errors.New("nil ach.File")
	}

	// For now we are inspecting the filepath to see if it came from our
	// configured reconciliation path. That's the best source of information
	// for when we should treat the file as a recon file.
	//
	// Example: /reconciliation/fileMoovTester_TRANACTIONSFAKE.TXT
	if !isReconciliationFile(pc.cfg, file) {
		return nil // skip the file
	}

	ctx, span := telemetry.StartSpan(ctx, "odfi-reconciliation-file", trace.WithAttributes(
		attribute.String("achgateway.filepath", file.Filepath),
	))
	defer span.End()

	// Record that we've encountered this ACH file
	creditReconciliationFilesProcessed.With(
		"origin", file.ACHFile.Header.ImmediateOrigin,
		"destination", file.ACHFile.Header.ImmediateDestination,
	).Add(1)
	logger = logger.With(log.Fields{
		"filepath": log.String(file.Filepath),
	})

	// Produce ReconciliationFile and/or ReconciliationEntry events from the downloaded reconciliation file
	if pc.cfg.ProduceFileEvents {
		logger.Log("odfi: producing reconciliation file event")

		err := pc.produceFileEvent(ctx, logger, file)
		if err != nil {
			return fmt.Errorf("producing file event: %w", err)
		}
	}
	if pc.cfg.ProduceEntryEvents {
		logger.Log("odfi: producing reconciliation entry events")

		err := pc.produceEntryEvents(ctx, logger, file)
		if err != nil {
			return fmt.Errorf("producing entry event: %w", err)
		}
	}

	return nil
}

func (pc *creditReconciliation) produceFileEvent(ctx context.Context, logger log.Logger, file File) error {
	var recons []models.Batch

	for i := range file.ACHFile.Batches {
		batch := models.Batch{
			Header: file.ACHFile.Batches[i].GetHeader(),
		}

		entries := file.ACHFile.Batches[i].GetEntries()
		batch.Entries = append(batch.Entries, entries...)

		if len(batch.Entries) > 0 {
			recons = append(recons, batch)
		}
	}
	if len(recons) > 0 {
		return pc.sendEvent(ctx, models.ReconciliationFile{
			Filename:        filepath.Base(file.Filepath),
			File:            file.ACHFile,
			Reconciliations: recons,
		})
	}
	return nil
}

func (pc *creditReconciliation) produceEntryEvents(ctx context.Context, logger log.Logger, file File) error {
	g := new(errgroup.Group)

	for i := range file.ACHFile.Batches {
		batch := file.ACHFile.Batches[i]

		entries := batch.GetEntries()
		for j := range entries {
			// Produce ReconciliationEntry event
			entry := entries[j]
			g.Go(func() error {
				return pc.sendEvent(ctx, models.ReconciliationEntry{
					Filename: filepath.Base(file.Filepath),
					Header:   batch.GetHeader(),
					Entry:    entry,
				})
			})
		}
	}

	return g.Wait()
}

func (pc *creditReconciliation) sendEvent(ctx context.Context, event interface{}) error {
	if pc.svc != nil {
		err := pc.svc.Send(ctx, models.Event{Event: event})
		if err != nil {
			return fmt.Errorf("sending reconciliations event: %w", err)
		}
	}
	return nil
}
