// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/moov-io/achgateway"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Slack struct {
	client     *http.Client
	webhookURL string
}

func NewSlack(cfg *service.Slack) (*Slack, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Slack{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		webhookURL: strings.TrimSpace(cfg.WebhookURL),
	}, nil
}

type uploadStatus string

const (
	success = uploadStatus("SUCCESSFUL")
	failed  = uploadStatus("FAILED")
)

func (s *Slack) Info(ctx context.Context, msg *Message) error {
	slackMsg := marshalSlackMessage(success, msg)
	return s.send(ctx, slackMsg)
}

func (s *Slack) Critical(ctx context.Context, msg *Message) error {
	slackMsg := marshalSlackMessage(failed, msg)
	return s.send(ctx, slackMsg)
}

func marshalSlackMessage(status uploadStatus, msg *Message) string {
	if msg.Contents != "" {
		return msg.Contents
	}

	slackMsg := fmt.Sprintf("%s %s of %s", status, msg.Direction, msg.Filename)
	if msg.Hostname != "" {
		if msg.Direction == Upload {
			slackMsg += " to " + msg.Hostname
		} else {
			slackMsg += " from " + msg.Hostname
		}
	}
	slackMsg += " with ODFI server\n"

	entries := countEntries(msg.File)
	debitTotal := convertDollar(msg.File.Control.TotalDebitEntryDollarAmountInFile)
	creditTotal := convertDollar(msg.File.Control.TotalCreditEntryDollarAmountInFile)
	slackMsg += fmt.Sprintf("%d Entries | Debits: %s | Credits: %s", entries, debitTotal, creditTotal)

	return slackMsg
}

type webhook struct {
	Text string `json:"text"`
}

func (s *Slack) send(ctx context.Context, msg string) error {
	_, span := telemetry.StartSpan(ctx, "notify-send-slack", trace.WithAttributes(
		attribute.String("achgateway.message", msg),
	))
	defer span.End()

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(&webhook{
		Text: msg,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("moov/achgateway %v slack notifier", achgateway.Version))

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
