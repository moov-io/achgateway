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
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/consul/api"

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

	TLS api.TLSConfig
}

type SessionConfig struct {
	CheckInterval time.Duration
}

// Client is a helpful wrapper around consul operations needed by achgateway.
// This client is not goroutine safe, so concurrnet calls are not supported.
type Client struct {
	cfg      *Config
	logger   log.Logger
	hostname string

	underlying *api.Client
	session    *Session
}

func NewConsulClient(logger log.Logger, config *Config) (*Client, error) {
	// Default settings we approve of
	consulClient, err := api.NewClient(&api.Config{
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

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to create consul client: %v", err)
	}

	client := &Client{
		cfg:        config,
		logger:     logger,
		underlying: consulClient,
		hostname:   hostname,
	}
	client.session, err = client.newSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create session: %v", err)
	}

	return client, nil
}

func (c *Client) Shutdown() {
	if c != nil {
		c.shutdownSession()
	}
}
