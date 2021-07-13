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

package compliance

import (
	"strings"
	"testing"
	"time"

	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"

	"github.com/stretchr/testify/require"
)

func TestCompliance(t *testing.T) {
	cfg := &models.TransformConfig{
		Encryption: &models.EncryptionConfig{
			AES: &models.AESConfig{
				Key: strings.Repeat("1", 16),
			},
		},
	}
	// randomly decide if we're going to base64 encode or not
	if time.Now().Unix()/2 == 0 {
		cfg.Encoding = &models.EncodingConfig{
			Base64: true,
		}
	}

	fileID, shardKey := base.ID(), base.ID()
	evt := models.Event{
		Event: models.FileUploaded{
			FileID:     fileID,
			ShardKey:   shardKey,
			Filename:   "20210709-0001.ach",
			UploadedAt: time.Now(),
		},
	}

	encrypted, err := Protect(cfg, evt)
	require.NoError(t, err)
	require.Greater(t, len(encrypted), 0)

	decrypted, err := Reveal(cfg, encrypted)
	require.NoError(t, err)
	require.Greater(t, len(decrypted), 0)

	var uploaded models.FileUploaded
	require.NoError(t, models.ReadEvent(decrypted, &uploaded))
	require.Equal(t, fileID, uploaded.FileID)
	require.Equal(t, shardKey, uploaded.ShardKey)
	require.Equal(t, "20210709-0001.ach", uploaded.Filename)
}
