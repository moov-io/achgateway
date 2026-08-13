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

package incoming

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQueueACHFileResponseJSON_OmitsOptionalTimes(t *testing.T) {
	bs, err := json.Marshal(QueueACHFileResponse{
		FileID:   "f1",
		ShardKey: "s1",
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"f1","shardKey":"s1","error":""}`, string(bs))
}

func TestQueueACHFileResponseJSON_IncludesOptionalTimes(t *testing.T) {
	next := time.Date(2025, time.January, 7, 16, 15, 0, 0, time.FixedZone("EST", -5*3600))
	accepted := time.Date(2025, time.January, 7, 18, 37, 0, 0, time.UTC)

	bs, err := json.Marshal(QueueACHFileResponse{
		FileID:     "f1",
		ShardKey:   "s1",
		NextCutoff: &next,
		AcceptedAt: &accepted,
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(bs, &decoded))
	require.Equal(t, "f1", decoded["id"])
	require.Equal(t, "s1", decoded["shardKey"])
	require.Equal(t, "", decoded["error"])
	require.Equal(t, "2025-01-07T16:15:00-05:00", decoded["nextCutoff"])
	require.Equal(t, "2025-01-07T18:37:00Z", decoded["acceptedAt"])
}
