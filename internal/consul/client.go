// generated-from:aac4f94179a969295e94b4572607e42b1419ca91e6a2c905c76717dc6a2f2525 DO NOT REMOVE, DO UPDATE

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

package consul

import (
	"time"

	consul "github.com/hashicorp/consul/api"

	"github.com/moov-io/base/log"
)

type Config struct {
	Address     string
	Scheme      string
	SessionPath string
	Tags        []string

	Token     string
	TokenFile string

	Datacenter string
	Namespace  string

	Session *SessionConfig

	TLS consul.TLSConfig
}

type SessionConfig struct {
	CheckInterval time.Duration
}

type Client struct {
	cfg        *Config
	underlying *consul.Client
}

func NewConsulClient(logger log.Logger, config *Config) (*Client, error) {
	// Default settings we approve of
	consulClient, err := consul.NewClient(&consul.Config{
		Address: config.Address,
		Scheme:  config.Scheme,

		Token:     config.Token,
		TokenFile: config.TokenFile,

		Datacenter: config.Datacenter,
		Namespace:  config.Namespace,

		TLSConfig: config.TLS,
	})
	if err != nil {
		return nil, logger.Fatal().LogErrorf("Error connecting to Consul (config: %v): %v", config, err).Err()
	}
	return &Client{
		cfg:        config,
		underlying: consulClient,
	}, nil
}
