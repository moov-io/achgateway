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
	"fmt"

	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
)

type streamService struct {
	topic *pubsub.Topic
}

func newStreamService(logger log.Logger, cfg *service.KafkaConfig) (*streamService, error) {
	topic, err := stream.Topic(logger, &service.Config{
		Inbound: service.Inbound{
			Kafka: cfg,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("events stream: %v", err)
	}
	return &streamService{
		topic: topic,
	}, nil
}

func (ss *streamService) Send(evt models.Event) error {
	err := ss.topic.Send(context.Background(), &pubsub.Message{
		Body: evt.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("error emitting %s: %v", evt.Type, err)
	}
	return nil
}
