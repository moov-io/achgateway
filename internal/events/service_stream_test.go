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
	"context"
	"testing"

	"github.com/moov-io/achgateway/internal/incoming/stream/streamtest"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"

	"github.com/stretchr/testify/require"
)

func TestStreamService(t *testing.T) {
	pub, sub := streamtest.InmemStream(t)
	svc := &streamService{topic: pub}

	shardKey, fileID := base.ID(), base.ID()
	err := svc.Send(context.Background(), models.Event{
		Event: models.FileUploaded{
			FileID:   fileID,
			ShardKey: shardKey,
		},
	})
	require.NoError(t, err)

	msg, err := sub.Receive(context.Background())
	require.NoError(t, err)
	msg.Ack()

	var body models.FileUploaded
	require.NoError(t, models.ReadEvent(msg.Body, &body))

	require.Equal(t, shardKey, body.ShardKey)
	require.Equal(t, fileID, body.FileID)
}
