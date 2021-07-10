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
	"encoding/json"

	"github.com/moov-io/achgateway/internal/crypt"
	"github.com/moov-io/achgateway/internal/encode"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
)

func Protect(cfg *service.TransformConfig, evt models.Event) ([]byte, error) {
	bs, err := json.Marshal(evt)
	if err != nil {
		return nil, err
	}
	// Return early if there are no encode/encrypt actions to take
	if cfg == nil {
		return bs, nil
	}

	// Encrypt
	cc, err := crypt.New(cfg.Encryption)
	if err != nil {
		return nil, err
	}
	bs, err = cc.Encrypt(bs)
	if err != nil {
		return nil, err
	}

	// Encode
	ec, err := encode.New(cfg.Encoding)
	if err != nil {
		return nil, err
	}
	bs, err = ec.Encode(bs)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

func Reveal(cfg *service.TransformConfig, data []byte) ([]byte, error) {
	if cfg == nil {
		return data, nil
	}

	// Decode
	ec, err := encode.New(cfg.Encoding)
	if err != nil {
		return nil, err
	}
	bs, err := ec.Decode(data)
	if err != nil {
		return nil, err
	}

	// Decrypt
	cc, err := crypt.New(cfg.Encryption)
	if err != nil {
		return nil, err
	}
	bs, err = cc.Decrypt(bs)
	if err != nil {
		return nil, err
	}

	return bs, nil
}
