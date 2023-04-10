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
	"errors"
	"fmt"

	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
)

type streamService struct {
	transformConfig *models.TransformConfig
	topic           *pubsub.Topic
}

func newStreamService(logger log.Logger, transformConfig *models.TransformConfig, cfg *service.EventsStream) (*streamService, error) {
	if cfg == nil {
		return nil, errors.New("nil EventsStream config")
	}

	topicConf := &service.Config{
		Inbound: service.Inbound{},
	}
	if cfg.InMem != nil {
		topicConf.Inbound.InMem = cfg.InMem
	}
	if cfg.Kafka != nil {
		topicConf.Inbound.Kafka = cfg.Kafka
	}

	topic, err := stream.Topic(logger, topicConf)
	if err != nil {
		return nil, fmt.Errorf("%T events stream: %v", topicConf.Inbound, err)
	}
	return &streamService{
		topic:           topic,
		transformConfig: transformConfig,
	}, nil
}

func (ss *streamService) Send(evt models.Event) error {
	bs, err := compliance.Protect(ss.transformConfig, evt)
	if err != nil {
		return err
	}
	err = ss.topic.Send(context.Background(), &pubsub.Message{
		Body: bs,
	})
	if err != nil {
		return fmt.Errorf("error emitting %s: %v", evt.Type, err)
	}
	return nil
}
