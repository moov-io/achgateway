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
	"strings"
	"time"

	"github.com/moov-io/ach"
)

var (
	// DefaultFilenameTemplate is achgateway's standard filename format for ACH files which are uploaded to an ODFI
	//
	// The format consists of a few parts: "year month day" timestamp, "hour minute" timestamp, and routing number
	//
	// Examples:
	//  - 20191010-0830-LiveODFI.ach      (.ShardName of "LiveODFI")
	//  - 20191010-0830-987654320.ach
	//  - 20191010-0830-987654320.ach.gpg (GPG encrypted)
	DefaultFilenameTemplate = `{{ date "20060102" }}-{{ date "150405" }}-{{ .RoutingNumber }}-{{ .Index }}.ach{{ if .GPG }}.gpg{{ end }}`
)

type Sharding struct {
	Shards   []Shard
	Mappings []ShardMapping
	Default  string
}

type ShardMapping struct {
	ShardKey  string
	ShardName string
}

func (cfg Sharding) Find(name string) *Shard {
	for i := range cfg.Shards {
		if strings.EqualFold(cfg.Shards[i].Name, name) {
			return &cfg.Shards[i]
		}
	}
	return nil
}

func (cfg Sharding) Validate() error {
	for i := range cfg.Shards {
		if err := cfg.Shards[i].Validate(); err != nil {
			return fmt.Errorf("shard[%d]: %v", i, err)
		}
	}
	return nil
}

type Shard struct {
	Name                     string
	Cutoffs                  Cutoffs
	PreUpload                *PreUpload
	UploadAgent              string
	Mergable                 MergableConfig
	OutboundFilenameTemplate string
	Output                   *Output
	Notifications            *Notifications
	Audit                    *AuditTrail
}

func (cfg Shard) Validate() error {
	if cfg.Name == "" {
		return errors.New("missing name")
	}
	if err := cfg.Cutoffs.Validate(); err != nil {
		return fmt.Errorf("cutoffs: %v", err)
	}
	if err := cfg.PreUpload.Validate(); err != nil {
		return fmt.Errorf("preupload: %v", err)
	}
	if cfg.UploadAgent == "" {
		return errors.New("missing upload agent")
	}
	if err := cfg.Output.Validate(); err != nil {
		return fmt.Errorf("output: %v", err)
	}
	if err := cfg.Notifications.Validate(); err != nil {
		return fmt.Errorf("notifications: %v", err)
	}
	if err := cfg.Audit.Validate(); err != nil {
		return fmt.Errorf("audit: %v", err)
	}
	return nil
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
		return errors.New("no windows")
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

type MergableConfig struct {
	Conditions     *ach.Conditions
	FlattenBatches *FlattenBatches
}

type FlattenBatches struct{}

type Output struct {
	Format string
}

func (cfg *Output) Validate() error {
	return nil
}

func (cfg *Shard) FilenameTemplate() string {
	if cfg == nil || cfg.OutboundFilenameTemplate == "" {
		return strings.TrimSpace(DefaultFilenameTemplate)
	}
	return strings.TrimSpace(cfg.OutboundFilenameTemplate)
}

func (m *ShardMapping) Validate() error {
	if m == nil {
		return nil
	}
	if m.ShardKey == "" {
		return errors.New("missing shard key")
	}
	if m.ShardName == "" {
		return errors.New("missing shard name")
	}
	return nil
}
