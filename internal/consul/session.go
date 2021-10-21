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
)

type Session struct {
	ID   string
	Name string
}

func (c *Client) newSession() (*Session, error) {
	seconds := 10.0
	if c.cfg.Session != nil {
		seconds = c.cfg.Session.CheckInterval.Seconds()
	}
	ttl := fmt.Sprintf("%.2fs", seconds)

	sessionID, _, err := c.underlying.Session().Create(&consul.SessionEntry{
		Name:     c.hostname,
		Behavior: "delete",
		TTL:      ttl,
	}, nil)

	if err != nil {
		return nil, c.logger.Fatal().LogErrorf("Error creating Consul Session for %s: %v", c.hostname, err).Err()
	}

	// make sure we renew the session
	go func() {
		doneChan := make(chan struct{})
		c.underlying.Session().RenewPeriodic(ttl, sessionID, nil, doneChan)
	}()

	return &Session{
		ID:   sessionID,
		Name: c.hostname,
	}, nil
}

func (c *Client) shutdownSession() {
	if c != nil && c.session != nil {
		c.underlying.Session().Destroy(c.session.ID, nil)
	}
}

func (c *Client) SessionID() string {
	if c == nil || c.session == nil {
		return c.session.ID
	}
	return ""
}

// ClearSession will attempt to wipe the existing session and create a new one.
// Often this is done as an attempt to resolve consul or network errors.
func (c *Client) ClearSession() error {
	if c == nil {
		return nil
	}
	c.shutdownSession()

	// Attempt a new session
	var err error
	c.session, err = c.newSession()
	if err != nil {
		return fmt.Errorf("unable to create session: %v", err)
	}
	return nil
}
