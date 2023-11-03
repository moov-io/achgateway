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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type incomingEmitter struct {
	svc   events.Emitter
	cfg   service.ODFIIncoming
	recon service.ODFIReconciliation
}

func IncomingEmitter(cfg service.ODFIIncoming, recon service.ODFIReconciliation, svc events.Emitter) *incomingEmitter {
	if !cfg.Enabled {
		return nil
	}
	return &incomingEmitter{
		svc:   svc,
		cfg:   cfg,
		recon: recon,
	}
}

func (pc *incomingEmitter) Type() string {
	return "incoming"
}

func (pc *incomingEmitter) Handle(ctx context.Context, logger log.Logger, file File) error {
	// Ignore files if they don't contain the PathMatcher value
	if pc.cfg.PathMatcher != "" && !strings.Contains(strings.ToLower(file.Filepath), pc.cfg.PathMatcher) {
		return nil // skip the file
	}

	// Skip files that have matched a previous Processor
	if pc.cfg.ExcludeCorrections && isCorrectionFile(file) {
		return nil
	}
	if pc.cfg.ExcludePrenotes && isPrenoteFile(file) {
		return nil
	}
	if pc.cfg.ExcludeReturns && isReturnFile(file) {
		return nil
	}
	if pc.cfg.ExcludeReconciliations && isReconciliationFile(pc.recon, file) {
		return nil
	}

	logger = logger.With(log.Fields{
		"filepath": log.String(file.Filepath),
	})

	// Skip when no ACH file was parsed
	if file.ACHFile == nil {
		logger.Warn().Log("no ACH file parsed")
		return nil
	}

	ctx, span := telemetry.StartSpan(ctx, "odfi-incoming-file", trace.WithAttributes(
		attribute.String("filepath", file.Filepath),
	))
	defer span.End()

	logger.Log("emitting IncomingFile event")
	err := pc.sendEvent(ctx, models.IncomingFile{
		Filename: filepath.Base(file.Filepath),
		File:     file.ACHFile,
	})
	return err
}

func (pc *incomingEmitter) sendEvent(ctx context.Context, event interface{}) error {
	if pc.svc != nil {
		err := pc.svc.Send(ctx, models.Event{Event: event})
		if err != nil {
			return fmt.Errorf("sending incoming file event: %w", err)
		}
	}
	return nil
}
