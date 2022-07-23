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

package rdfi

import (
	"path/filepath"
	"strings"

	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"
)

type incomingEmitter struct {
	logger log.Logger
	svc    events.Emitter

	cfg   service.ODFIIncoming
	recon service.ODFIReconciliation
}

func IncomingEmitter(logger log.Logger, cfg service.ODFIIncoming, recon service.ODFIReconciliation, svc events.Emitter) *incomingEmitter {
	if !cfg.Enabled {
		return nil
	}
	return &incomingEmitter{
		logger: logger,
		svc:    svc,
		cfg:    cfg,
		recon:  recon,
	}
}

func (pc *incomingEmitter) Type() string {
	return "incoming"
}

func (pc *incomingEmitter) Handle(file File) error {
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

	pc.logger.With(log.Fields{
		"filepath": log.String(file.Filepath),
	}).Log("emitting IncomingFile event")

	pc.sendEvent(models.IncomingFile{
		Filename: filepath.Base(file.Filepath),
		File:     file.ACHFile,
	})

	return nil
}

func (pc *incomingEmitter) sendEvent(event interface{}) {
	if pc.svc != nil {
		err := pc.svc.Send(models.Event{Event: event})
		if err != nil {
			pc.logger.Logf("error sending pre-note event: %v", err)
		}
	}
}
