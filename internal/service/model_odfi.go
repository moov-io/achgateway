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
	"fmt"
	"time"

	"github.com/moov-io/achgateway/internal/storage"
)

type ODFIConfig struct {
	Origination OriginationConfig
	Listen      ListenConfig
}

func (cfg ODFIConfig) Validate() error {
	if err := cfg.Origination.Validate(); err != nil {
		return fmt.Errorf("origination: %v", err)
	}
	if err := cfg.Listen.Validate(); err != nil {
		return fmt.Errorf("listen: %v", err)
	}
	return nil
}

type OriginationConfig struct {
	Kafka   *KafkaConfig
	Merging Merging
	Audit   *AuditTrail
}

func (cfg OriginationConfig) Validate() error {
	if err := cfg.Kafka.Validate(); err != nil {
		return fmt.Errorf("kafka: %v", err)
	}
	return nil
}

type Merging struct {
	Storage storage.Config
}

type ListenConfig struct {
	Processors ODFIProcessors
	Interval   time.Duration
	ShardNames []string
	Storage    LocalStorage
	Audit      *AuditTrail
	Events     EventsConfig
}

func (cfg ListenConfig) Validate() error {
	return nil
}

type ODFIProcessors struct {
	Corrections    ODFICorrections
	Reconciliation ODFIReconciliation
	Returns        ODFIReturns
}

type ODFICorrections struct {
	Enabled     bool
	PathMatcher string
}

type ODFIReconciliation struct {
	Enabled     bool
	PathMatcher string
}

type ODFIReturns struct {
	Enabled     bool
	PathMatcher string
}
