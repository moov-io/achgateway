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

package crypt

import (
	"strings"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCryptor__AES(t *testing.T) {
	cc, err := New(&service.EncryptionConfig{
		AES: &service.AESConfig{
			Key: strings.Repeat("1", 16),
		},
	})
	require.NoError(t, err)

	enc, err := cc.Encrypt([]byte("hello, world"))
	require.NoError(t, err)
	require.Greater(t, len(enc), 0)

	dec1, err := cc.Decrypt(enc)
	require.NoError(t, err)
	require.Equal(t, "hello, world", string(dec1))

	dec2, err := cc.Decrypt(enc)
	require.NoError(t, err)
	require.Equal(t, "hello, world", string(dec2))
}
