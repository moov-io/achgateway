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

package service

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/moov-io/achgateway/internal/mask"
	"github.com/moov-io/base/strx"
)

type AuditTrail struct {
	ID        string
	BucketURI string
	BasePath  string // e.g. 'incoming' or 'outgoing'
	GPG       *GPG
}

func (cfg *AuditTrail) Validate() error {
	if cfg == nil {
		return nil
	}
	if cfg.BucketURI == "" {
		return errors.New("missing bucket_uri")
	}
	return nil
}

type GPG struct {
	KeyFile string
	Signer  *Signer
}

type Signer struct {
	KeyFile     string
	KeyPassword string
}

func (cfg *Signer) Password() string {
	return strx.Or(os.Getenv("PIPELINE_SIGNING_KEY_PASSWORD"), cfg.KeyPassword)
}

func (cfg *Signer) MarshalJSON() ([]byte, error) {
	type Aux struct {
		KeyFile     string
		KeyPassword string
	}
	return json.Marshal(Aux{
		KeyFile:     cfg.KeyFile,
		KeyPassword: mask.Password(cfg.Password()),
	})
}
