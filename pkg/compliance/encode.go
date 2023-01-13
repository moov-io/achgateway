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
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/moov-io/achgateway/pkg/models"
)

type coder interface {
	Encode(data []byte) ([]byte, error)
	Decode(data []byte) ([]byte, error)
}

func newCoder(cfg *models.EncodingConfig) (coder, error) {
	switch {
	case cfg == nil:
		return &mockCoder{}, nil
	case cfg.Compress:
		return &gzipCoder{}, nil
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

type gzipCoder struct{}

func (*gzipCoder) Encode(data []byte) ([]byte, error) {
	var out bytes.Buffer
	w := gzip.NewWriter(&out)
	_, err := w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("gzip encode: %v", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %v", err)
	}
	return out.Bytes(), nil
}

func (*gzipCoder) Decode(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		if err == gzip.ErrHeader {
			return data, nil
		}
		return nil, fmt.Errorf("gzip decoder: %v", err)
	}
	bs, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("gzip readall: %v", err)
	}
	return bs, nil
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
