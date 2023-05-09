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

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/pkg/models"
)

type Inbound struct {
	HTTP  HTTPConfig
	InMem *InMemory
	Kafka *KafkaConfig
	ODFI  *ODFIFiles
	Audit *AuditTrail
}

func (cfg Inbound) Validate() error {
	if err := cfg.InMem.Validate(); err != nil {
		return fmt.Errorf("inmem: %v", err)
	}
	if err := cfg.Kafka.Validate(); err != nil {
		return fmt.Errorf("kafka: %v", err)
	}
	if err := cfg.ODFI.Validate(); err != nil {
		return fmt.Errorf("odfi: %v", err)
	}
	return nil
}

type HTTPConfig struct {
	BindAddress string
	TLS         TLSConfig

	Transform    *models.TransformConfig
	MaxBodyBytes int64
}

type InMemory struct {
	URL string
}

func (cfg *InMemory) Validate() error {
	if cfg != nil && cfg.URL == "" {
		return errors.New("missing URL")
	}
	return nil
}

type KafkaConfig struct {
	Brokers []string
	Key     string
	Secret  string
	Group   string
	Topic   string
	TLS     bool

	// AutoCommit in Sarama refers to "automated publishing of consumer offsets
	// to the broker" rather than a Kafka broker's meaning of "commit consumer
	// offsets on read" which leads to "at-most-once" delivery.
	AutoCommit bool

	Consumer KafkaConsumerConfig
	Producer KafkaProducerConfig

	Transform *models.TransformConfig
}

// KafkaConsumerConfig represents config options from sarama's Config.Consumer type.
// When these fields are non-zero values they will be set on the pubsub Subscription
type KafkaConsumerConfig struct{}

// KafkaProducerConfig represents config options from sarama's Config.Producer type.
// When these fields are non-zero values they will be set on the pubsub Topic
type KafkaProducerConfig struct {
	MaxMessageBytes int
}

func (cfg *KafkaConfig) Validate() error {
	if cfg == nil {
		return nil
	}
	if len(cfg.Brokers) == 0 {
		return errors.New("missing brokers")
	}
	if cfg.Topic == "" {
		return errors.New("missing topic")
	}
	return nil
}

type ODFIFiles struct {
	Processors ODFIProcessors
	Interval   time.Duration
	ShardNames []string
	Storage    ODFIStorage
	Audit      *AuditTrail
}

func (cfg *ODFIFiles) Validate() error {
	if cfg == nil {
		return nil
	}
	if err := cfg.Processors.Validate(); err != nil {
		return fmt.Errorf("processors: %v", err)
	}
	if cfg.Interval <= 0*time.Second {
		return errors.New("invalid interval")
	}
	if len(cfg.ShardNames) == 0 {
		return errors.New("missing shard names")
	}
	return nil
}

type ODFIProcessors struct {
	Corrections    ODFICorrections
	Incoming       ODFIIncoming
	Reconciliation ODFIReconciliation
	Prenotes       ODFIPrenotes
	Returns        ODFIReturns

	Validation ach.ValidateOpts
}

func (cfg ODFIProcessors) Validate() error {
	return nil
}

type ODFICorrections struct {
	Enabled     bool
	PathMatcher string
}

type ODFIIncoming struct {
	Enabled     bool
	PathMatcher string

	ExcludeCorrections     bool
	ExcludePrenotes        bool
	ExcludeReturns         bool
	ExcludeReconciliations bool
}

type ODFIReconciliation struct {
	Enabled     bool
	PathMatcher string
}

type ODFIPrenotes struct {
	Enabled     bool
	PathMatcher string
}

type ODFIReturns struct {
	Enabled     bool
	PathMatcher string
}

type ODFIStorage struct {
	// Directory is the local filesystem path for downloading files into
	Directory string

	// CleanupLocalDirectory determines if we delete the local directory after
	// processing is finished. Leaving these files around helps debugging, but
	// also exposes customer information.
	CleanupLocalDirectory bool

	// KeepRemoteFiles determines if we delete the remote file on an ODFI's server
	// after downloading and processing of each file.
	KeepRemoteFiles bool

	// RemoveZeroByteFiles determines if we should delete files that are zero bytes
	RemoveZeroByteFiles bool
}
