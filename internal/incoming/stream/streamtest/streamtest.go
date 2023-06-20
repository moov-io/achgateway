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

package streamtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
	"gocloud.dev/pubsub"
)

func InmemStream(t *testing.T) (stream.Publisher, stream.Subscription) {
	t.Helper()

	n, _ := rand.Int(rand.Reader, big.NewInt(10000))
	conf := &service.Config{
		Inbound: service.Inbound{
			InMem: &service.InMemory{
				URL: fmt.Sprintf("mem://achgateway-%s-%d", t.Name(), n),
			},
		},
	}
	topic, err := stream.Topic(log.NewTestLogger(), conf)
	require.NoError(t, err)

	sub, err := stream.OpenSubscription(log.NewTestLogger(), conf)
	require.NoError(t, err)
	t.Cleanup(func() { sub.Shutdown(context.Background()) })

	return topic, sub
}

type FailedSubscription struct {
	Err error
}

func (s *FailedSubscription) Receive(ctx context.Context) (*pubsub.Message, error) {
	return nil, s.Err
}

func (s *FailedSubscription) Shutdown(ctx context.Context) error {
	return nil
}

func FailingSubscription(err error) *FailedSubscription {
	return &FailedSubscription{Err: err}
}
