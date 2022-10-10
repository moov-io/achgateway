// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"fmt"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/compliance"
	"github.com/moov-io/achgateway/pkg/kafka"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"
	"gocloud.dev/pubsub"
)

type Kafka struct {
	publisher *pubsub.Topic
	cfg       *service.KafkaConfig
}

func NewKafka(logger log.Logger, cfg *service.KafkaConfig) (*Kafka, error) {
	publisher, err := kafka.OpenKafkaTopic(logger, cfg.Inbound.Kafka)
	if err != nil {
		return nil, err
	}

	return &Kafka{publisher: publisher, cfg: cfg}, nil
}

type event struct {
	Direction    Direction    `json:"direction"`
	FileName     string       `json:"fileName"`
	Entries      int          `json:"entries"`
	DebitTotal   string       `json:"debitTotal"`
	CreditTotal  string       `json:"creditTotal"`
	Hostname     string       `json:"hostname"`
	UploadStatus uploadStatus `json:"uploadStatus"`
}

func (s *Kafka) Info(msg *Message) error {
	event := marshalKafkaMessage(success, msg)
	return s.send(event)
}

func (s *Kafka) Critical(msg *Message) error {
	event := marshalKafkaMessage(failed, msg)
	return s.send(event)
}

func marshalKafkaMessage(status uploadStatus, msg *Message) event {
	entries := countEntries(msg.File)
	debitTotal := convertDollar(msg.File.Control.TotalDebitEntryDollarAmountInFile)
	creditTotal := convertDollar(msg.File.Control.TotalCreditEntryDollarAmountInFile)

	return event{
		UploadStatus: status,
		Direction:    msg.Direction,
		FileName:     msg.Filename,
		Entries:      entries,
		DebitTotal:   debitTotal,
		CreditTotal:  creditTotal,
		Hostname:     msg.Hostname,
	}
}

func (s *Kafka) send(evt event) error {
	bs, err := compliance.Protect(s.cfg.Transform, models.Event{
		Type:  "",
		Event: evt,
	})
	if err != nil {
		return fmt.Errorf("unable to protect notifer kafka event: %v", err)
	}

	return s.publisher.Send(context.Background(), &pubsub.Message{
		Body: bs,
	})
}
