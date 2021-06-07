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
	"fmt"
	"net/url"
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/hashicorp/go-retryablehttp"
)

type webService struct {
	cfg      service.WebhookConfig
	client   *retryablehttp.Client
	endpoint *url.URL
	logger   log.Logger
}

func newWebhookService(logger log.Logger, cfg *service.WebhookConfig) (*webService, error) {
	if cfg == nil || cfg.Endpoint == "" {
		return nil, nil
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("webhook: %v", err)
	}
	return &webService{
		cfg:      *cfg,
		client:   retryablehttp.NewClient(),
		endpoint: u,
		logger:   logger,
	}, nil
}

func (w *webService) FilesUploaded(shardKey string, fileIDs []string) error {
	for i := range fileIDs {
		msg := FileUploaded{
			FileID:     fileIDs[i],
			ShardKey:   shardKey,
			UploadedAt: time.Now(),
		}
		req, err := retryablehttp.NewRequest("POST", w.endpoint.String(), bytes.NewReader(msg.Bytes()))
		if err != nil {
			return fmt.Errorf("error preparing request: %v", err)
		}
		resp, err := w.client.Do(req)
		if err != nil {
			w.logger.Info().Logf("problem sending fileID=%s webhook: %v", fileIDs[i], err)
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
	}
	return nil
}
