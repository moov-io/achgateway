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
)

var (
	returnEntriesProcessed = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "return_entries_processed",
		Help: "Counter of return EntryDetail records processed",
	}, []string{"origin", "destination", "code"})
)

type returnEmitter struct {
	svc events.Emitter
	cfg service.ODFIReturns
}

func ReturnEmitter(cfg service.ODFIReturns, svc events.Emitter) *returnEmitter {
	if !cfg.Enabled {
		return nil
	}
	return &returnEmitter{
		svc: svc,
		cfg: cfg,
	}
}

func (pc *returnEmitter) Type() string {
	return "return"
}

func isReturnFile(file File) bool {
	return len(file.ACHFile.ReturnEntries) >= 0
}

func (pc *returnEmitter) Handle(ctx context.Context, logger log.Logger, file File) error {
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

	logger = logger.With(log.Fields{
		"origin":      log.String(file.ACHFile.Header.ImmediateOrigin),
		"destination": log.String(file.ACHFile.Header.ImmediateDestination),
	})
	logger.Log("odfi: processing return file")

	ctx, span := telemetry.StartSpan(ctx, "odfi-return-file", trace.WithAttributes(
		attribute.String("filename", file.Filepath),
		attribute.Int("return_entries", len(file.ACHFile.ReturnEntries)),
	))
	defer span.End()

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

			logger.With(log.Fields{
				"origin":      log.String(file.ACHFile.Header.ImmediateOrigin),
				"destination": log.String(file.ACHFile.Header.ImmediateDestination),
			}).Log(fmt.Sprintf("odfi: return batch %d entry %d code %s", i, j, returnCode.Code))
		}
	}
	return pc.sendEvent(ctx, msg)
}

func (pc *returnEmitter) sendEvent(ctx context.Context, event interface{}) error {
	if pc.svc != nil {
		err := pc.svc.Send(ctx, models.Event{Event: event})
		if err != nil {
			return fmt.Errorf("sending return event: %w", err)
		}
	}
	return nil
}
