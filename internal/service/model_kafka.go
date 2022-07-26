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

	"github.com/moov-io/achgateway/pkg/models"
)

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

	Transform *models.TransformConfig
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
