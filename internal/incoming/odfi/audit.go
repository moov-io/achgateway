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
	"io"

	"github.com/moov-io/achgateway/internal/audittrail"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/strx"
)

type AuditSaver struct {
	storage  audittrail.Storage
	basePath string
	hostname string
}

func (as *AuditSaver) save(filepath string, data io.Reader) error {
	if as == nil {
		return nil
	}
	return as.storage.SaveFileStream(filepath, data)
}

func newAuditSaver(hostname string, cfg *service.AuditTrail) (*AuditSaver, error) {
	if cfg == nil {
		return nil, nil
	}

	storage, err := audittrail.NewStorage(cfg)
	if err != nil {
		return nil, fmt.Errorf("odfi: audit: %v", err)
	}

	return &AuditSaver{
		storage:  storage,
		basePath: strx.Or(cfg.BasePath, "odfi"),
		hostname: hostname,
	}, nil
}
