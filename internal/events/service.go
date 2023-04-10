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

package events

import (
	"errors"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"
)

type Emitter interface {
	Send(evt models.Event) error
}

func NewEmitter(logger log.Logger, cfg *service.EventsConfig) (Emitter, error) {
	if cfg == nil {
		return &MockEmitter{}, nil
	}
	if cfg.Stream != nil {
		return newStreamService(logger, cfg.Transform, cfg.Stream)
	}
	if cfg.Webhook != nil {
		return newWebhookService(logger, cfg.Transform, cfg.Webhook)
	}
	return nil, errors.New("unknown events config")
}

type MockEmitter struct{}

func (*MockEmitter) Send(evt models.Event) error {
	return nil
}
