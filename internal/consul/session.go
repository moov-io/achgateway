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

	consul "github.com/hashicorp/consul/api"

	"github.com/moov-io/base/log"
)

type Session struct {
	ID   string
	Name string
}

func NewSession(logger log.Logger, consulClient Client, shardName string) (*Session, error) {
	sessionName := consulClient.Cfg.SessionPath + shardName
	sessionID, _, err := consulClient.ConsulClient.Session().Create(&consul.SessionEntry{
		Name:     sessionName,
		Behavior: "delete",
		TTL:      fmt.Sprintf("%.2fs", consulClient.Cfg.HealthCheckInterval.Seconds()),
	}, nil)

	if err != nil {
		return nil, logger.Fatal().LogErrorf("Error creating Consul Session for %s: %v", sessionName, err).Err()
	}

	return &Session{
		ID:   sessionID,
		Name: sessionName,
	}, nil
}
