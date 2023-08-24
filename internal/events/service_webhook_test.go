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
	"io"
	"net/http"
	"testing"

	"github.com/moov-io/achgateway/internal/admintest"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestWebhookService(t *testing.T) {
	adminServer := admintest.Server(t)

	var body *models.FileUploaded
	adminServer.AddHandler("/hook", func(w http.ResponseWriter, r *http.Request) {
		bs, _ := io.ReadAll(r.Body)

		var wrapper models.FileUploaded
		if err := models.ReadEvent(bs, &wrapper); err != nil {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			body = &wrapper
			w.WriteHeader(http.StatusOK)
		}
	})

	svc, err := newWebhookService(log.NewTestLogger(), nil, &service.WebhookConfig{
		Endpoint: "http://" + adminServer.BindAddr() + "/hook",
	})
	require.NoError(t, err)

	shardKey, fileID := base.ID(), base.ID()
	err = svc.Send(models.Event{
		Event: models.FileUploaded{
			FileID:   fileID,
			ShardKey: shardKey,
		},
	})
	require.NoError(t, err)

	require.Equal(t, shardKey, body.ShardKey)
	require.Equal(t, fileID, body.FileID)
}
