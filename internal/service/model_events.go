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

type EventsConfig struct {
	Stream    *EventsStream
	Webhook   *WebhookConfig
	Transform *models.TransformConfig
}

func (cfg *EventsConfig) Validate() error {
	if cfg == nil {
		return nil
	}
	if err := cfg.Stream.Validate(); err != nil {
		return err
	}
	if err := cfg.Webhook.Validate(); err != nil {
		return err
	}
	return nil
}

type EventsStream struct {
	InMem *InMemory
	Kafka *KafkaConfig
}

func (cfg *EventsStream) Validate() error {
	if cfg == nil {
		return nil
	}
	if err := cfg.Kafka.Validate(); err != nil {
		return err
	}
	return nil
}

type WebhookConfig struct {
	Endpoint string
}

func (cfg *WebhookConfig) Validate() error {
	if cfg == nil {
		return nil
	}
	if cfg.Endpoint == "" {
		return errors.New("missing endpoint")
	}
	return nil
}
