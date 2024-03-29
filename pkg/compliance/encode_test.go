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
	"unicode/utf8"

	"github.com/moov-io/achgateway/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestCoder(t *testing.T) {
	ec, err := newCoder(&models.EncodingConfig{
		Base64: true,
	})
	require.NoError(t, err)

	enc, err := ec.Encode([]byte("hello, world"))
	require.NoError(t, err)
	require.Equal(t, "aGVsbG8sIHdvcmxk", string(enc))

	dec, err := ec.Decode(enc)
	require.NoError(t, err)
	require.Equal(t, "hello, world", string(dec))
}

func TestGzipCoder(t *testing.T) {
	ec, err := newCoder(&models.EncodingConfig{
		Compress: true,
	})
	require.NoError(t, err)

	input := strings.Repeat("1234567890", 21)
	expectedLength := utf8.RuneCountInString(input)

	encoded, err := ec.Encode([]byte(input))
	require.NoError(t, err)
	require.Len(t, encoded, 37)

	decoded, err := ec.Decode(encoded)
	require.NoError(t, err)
	require.Len(t, decoded, expectedLength)
	require.Equal(t, input, string(decoded))

	// .Decode should skip gzip if the input isn't compressed
	decoded, err = ec.Decode([]byte(input))
	require.NoError(t, err)
	require.Len(t, string(decoded), expectedLength)
	require.Equal(t, input, string(decoded))
}
