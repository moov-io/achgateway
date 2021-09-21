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
	"sync"
	"time"

	"github.com/moov-io/base/log"
)

type Wrapper struct {
	logger log.Logger
	client *Client

	mu         sync.Mutex
	sessions   map[string]*Session // shardKey is the key
	leadership map[string]time.Time
}

func NewWrapper(logger log.Logger, client *Client) *Wrapper {
	w := &Wrapper{
		logger:     logger,
		client:     client,
		sessions:   make(map[string]*Session),
		leadership: make(map[string]time.Time),
	}
	return w
}

func (w *Wrapper) Acquire(shardKey string) (isLeader bool, err error) {
	if w == nil {
		return true, nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.leadership[shardKey]; exists {
		return true, nil
	}

	// Find an active session
	sesh, exists := w.sessions[shardKey]
	if sesh == nil || !exists {
		ss, err := NewSession(w.logger, w.client, shardKey)
		if err != nil {
			return false, nil
		}
		w.sessions[shardKey] = ss
		sesh = ss
	}

	// Attempt to grab a lock
	if err := AcquireLock(w.logger, w.client, sesh); err != nil {
		// not the leader
		return false, nil
	}

	// we are the leader
	w.leadership[shardKey] = time.Now()
	return true, nil
}

func (w *Wrapper) Shutdown() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, sesh := range w.sessions {
		w.client.underlying.Session().Destroy(sesh.ID, nil)
	}
}
