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
	"strings"

	"github.com/moov-io/base/log"

	consulapi "github.com/hashicorp/consul/api"
)

func (c *Client) AcquireLock(key string) error {
	isLeader, _, err := c.underlying.KV().Acquire(&consulapi.KVPair{
		Key:     key,
		Value:   []byte(c.session.ID),
		Session: c.session.ID,
	}, nil)
	if err != nil {
		return err
	}
	if isLeader {
		return nil
	}
	return c.logger.Info().With(log.Fields{
		"hostname":  log.String(c.hostname),
		"sessionID": log.String(c.session.ID),
	}).LogErrorf("we are not the leader of %s", key).Err()
}

func AcquireLock(logger log.Logger, client *Client, leaderKey string) error {
	if client == nil {
		return nil // no leader defined, skip election
	}

	var lockErr error

	var try func(attempts int, leaderKey string) error
	try = func(attempts int, leaderKey string) error {
		if attempts >= 3 {
			return fmt.Errorf("too many retries: %v", lockErr)
		}
		attempts++

		// Attempt writing to the KV path
		lockErr := client.AcquireLock(leaderKey)

		if lockErr != nil {
			// IsRetryableError returns true for 500 errors from the Consul servers, and network connection errors.
			// These errors are not retryable for writes (which is what AcquireLock performs).
			if consulapi.IsRetryableError(lockErr) || strings.Contains(lockErr.Error(), "invalid session") {
				// If we're able to create a new session and see if achgateway can continue on.
				// This error will be bubbled up to our Alterer to notify humans.
				if innerErr := client.ClearSession(); innerErr != nil {
					return fmt.Errorf("really bad consul error: %v and unable to restart session %v", lockErr, innerErr)
				} else {
					logger.Info().With(log.Fields{
						"sessionID": log.String(client.SessionID()),
					}).Logf("started new session")
				}
			}

			// Retry leadership attempt
			return try(attempts, leaderKey)
		}
		// We've got an active session and leadership of the shard
		return nil
	}

	return try(0, leaderKey)
}
