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

package encode

import (
	"bytes"
	"encoding/base64"
	"errors"

	"github.com/moov-io/achgateway/internal/service"
)

type Coder interface {
	Encode(data []byte) ([]byte, error)
	Decode(data []byte) ([]byte, error)
}

func New(cfg *service.EncodingConfig) (Coder, error) {
	switch {
	case cfg == nil:
		return &mockCoder{}, nil

	case cfg.Base64:
		return &base64Coder{}, nil
	}
	return nil, errors.New("unknown coding")
}

type mockCoder struct{}

func (*mockCoder) Encode(data []byte) ([]byte, error) {
	return data, nil
}

func (*mockCoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}

type base64Coder struct{}

func (*base64Coder) Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(base64.RawStdEncoding.EncodeToString(data))
	return buf.Bytes(), nil
}

func (*base64Coder) Decode(data []byte) ([]byte, error) {
	dst := make([]byte, base64.RawStdEncoding.DecodedLen(len(data)))
	_, err := base64.RawStdEncoding.Decode(dst, data)
	if err != nil {
		return nil, err
	}
	return dst, nil
}
