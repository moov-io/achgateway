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
	"bytes"
	"context"
	"fmt"
	"net/url"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/hashicorp/go-retryablehttp"
)

type webhookService struct {
	cfg             service.WebhookConfig
	transformConfig *models.TransformConfig
	client          *retryablehttp.Client
	endpoint        *url.URL
	logger          log.Logger
}

func newWebhookService(logger log.Logger, transformConfig *models.TransformConfig, cfg *service.WebhookConfig) (*webhookService, error) {
	if cfg == nil || cfg.Endpoint == "" {
		return nil, nil
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("webhook: %v", err)
	}
	return &webhookService{
		cfg:             *cfg,
		transformConfig: transformConfig,
		client:          retryablehttp.NewClient(),
		endpoint:        u,
		logger:          logger,
	}, nil
}

func (w *webhookService) Send(_ context.Context, evt models.Event) error {
	bs, err := compliance.Protect(w.transformConfig, evt)
	if err != nil {
		return err
	}

	req, err := retryablehttp.NewRequest("POST", w.endpoint.String(), bytes.NewReader(bs))
	if err != nil {
		return fmt.Errorf("error preparing request: %v", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Info().Logf("problem sending event: %v", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	return nil
}
