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

package service

import (
	"errors"
	"fmt"
	"time"
)

var (
	// DefaultFilenameTemplate is paygate's standard filename format for ACH files which are uploaded to an ODFI
	//
	// The format consists of a few parts: "year month day" timestamp, "hour minute" timestamp, and routing number
	//
	// Examples:
	//  - 20191010-0830-987654320.ach
	//  - 20191010-0830-987654320.ach.gpg (GPG encrypted)
	DefaultFilenameTemplate = `{{ date "20060102" }}-{{ date "1504" }}-{{ .RoutingNumber }}.ach{{ if .GPG }}.gpg{{ end }}`
)

type Shards []Shard

func (cfg Shards) Validate() error {
	for i := range cfg {
		if err := cfg[i].Validate(); err != nil {
			return fmt.Errorf("shard[%d]: %v", i, err)
		}
	}
	return nil
}

type Shard struct {
	Name                     string
	Cutoffs                  Cutoffs
	PreUpload                PreUpload
	Upload                   UploadAgents
	OutboundFilenameTemplate string
	Output                   Output
	Notifications            Notifications
	Audit                    AuditTrail
}

func (cfg Shard) Validate() error {
	return nil // TODO(adam):
}

type Cutoffs struct {
	Timezone string
	Windows  []string
}

func (cfg Cutoffs) Location() *time.Location {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil
	}
	return loc
}

func (cfg Cutoffs) Validate() error {
	if loc := cfg.Location(); loc == nil {
		return fmt.Errorf("unknown Timezone=%q", cfg.Timezone)
	}
	if len(cfg.Windows) == 0 {
		return errors.New("no cutoff windows")
	}
	return nil
}

type PreUpload struct {
	GPG *GPG
}

func (cfg *PreUpload) Validate() error {
	if cfg == nil {
		return nil
	}
	if cfg.GPG != nil && cfg.GPG.KeyFile == "" {
		return errors.New("gpg: missing key file")
	}
	return nil
}

type Output struct {
	Format string
}

func (cfg *Output) Validate() error {
	return nil
}

func (cfg *Shard) FilenameTemplate() string {
	if cfg == nil || cfg.OutboundFilenameTemplate == "" {
		return DefaultFilenameTemplate
	}
	return cfg.OutboundFilenameTemplate
}
