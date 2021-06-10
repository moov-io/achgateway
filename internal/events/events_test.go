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

package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/base"

	"github.com/stretchr/testify/require"
)

func TestEvent(t *testing.T) {
	check := func(t *testing.T, inner interface{}, matchers ...string) {
		t.Helper()
		evt := Event{
			Event: inner,
		}
		bs, err := json.Marshal(evt)
		require.NoError(t, err)
		for i := range matchers {
			require.Contains(t, string(bs), matchers[i])
		}
	}
	// Verfiy every event type
	check(t, CorrectionFile{
		File: ach.NewFile(),
	}, `"type":"CorrectionFile"`)

	check(t, IncomingFile{
		File: ach.NewFile(),
	}, `"type":"IncomingFile"`)

	check(t, ReturnFile{
		File: ach.NewFile(),
	}, `"type":"ReturnFile"`)

	check(t, FileUploaded{
		FileID:     base.ID(),
		ShardKey:   base.ID(),
		UploadedAt: time.Now(),
	}, `"type":"FileUploaded"`)
}
