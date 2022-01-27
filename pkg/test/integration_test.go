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

package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"

	"github.com/stretchr/testify/require"
)

func TestReadEvent__ValidateOpts(t *testing.T) {
	file, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)

	queueEvent := models.QueueACHFile{
		FileID:   base.ID(),
		ShardKey: "testing",
		File:     file,
	}
	queueEvent.SetValidation(&ach.ValidateOpts{
		BypassOriginValidation:      true,
		BypassDestinationValidation: true,
	})

	evt := models.Event{
		Event: queueEvent,
	}

	cfg := &models.TransformConfig{
		Encryption: &models.EncryptionConfig{
			AES: &models.AESConfig{
				Key: strings.Repeat("1", 16),
			},
		},
		Encoding: &models.EncodingConfig{
			Base64: true,
		},
	}

	bs, err := compliance.Protect(cfg, evt)
	require.NoError(t, err)

	bs, err = compliance.Reveal(cfg, bs)
	require.NoError(t, err)

	var inc incoming.ACHFile
	err = models.ReadEvent(bs, &inc)
	require.NoError(t, err)

	require.Equal(t, queueEvent.FileID, inc.FileID)
	require.Equal(t, queueEvent.ShardKey, inc.ShardKey)

	opts := inc.File.GetValidation()
	require.True(t, opts.BypassOriginValidation)
	require.True(t, opts.BypassDestinationValidation)
	require.False(t, opts.CustomTraceNumbers)
}
