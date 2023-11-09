// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/sethvargo/go-retry"
)

// MultiSender is a Sender which will attempt to send each Message to every
// included Sender and returns the first error encountered.
type MultiSender struct {
	logger      log.Logger
	senders     []Sender
	retryConfig *service.NotificationRetries
}

func NewMultiSender(logger log.Logger, cfg *service.Notifications, notifiers *service.UploadNotifiers) (*MultiSender, error) {
	ms := &MultiSender{logger: logger}
	if cfg == nil || notifiers == nil {
		return ms, nil
	}
	if cfg.Retry != nil {
		ms.retryConfig = cfg.Retry
	}

	emails := cfg.FindEmails(notifiers.Email)
	for i := range emails {
		sender, err := NewEmail(&emails[i])
		if err != nil {
			return nil, err
		}
		ms.senders = append(ms.senders, sender)
	}

	pds := cfg.FindPagerDutys(notifiers.PagerDuty)
	for i := range pds {
		sender, err := NewPagerDuty(&pds[i])
		if err != nil {
			return nil, err
		}
		ms.senders = append(ms.senders, sender)
	}

	slacks := cfg.FindSlacks(notifiers.Slack)
	for i := range slacks {
		sender, err := NewSlack(&slacks[i])
		if err != nil {
			return nil, err
		}
		ms.senders = append(ms.senders, sender)
	}

	return ms, nil
}

func setupBackoff(cfg *service.NotificationRetries) (retry.Backoff, error) {
	fib := retry.NewFibonacci(cfg.Interval)
	if fib == nil {
		return nil, fmt.Errorf("problem creating %v as fibonacci", cfg.Interval)
	}
	fib = retry.WithMaxRetries(cfg.MaxRetries, fib)
	return fib, nil
}

func (ms *MultiSender) senderTypes() string {
	var out []string
	for i := range ms.senders {
		out = append(out, fmt.Sprintf("%T", ms.senders[i]))
	}
	return strings.Join(out, ", ")
}

func (ms *MultiSender) Info(ctx context.Context, msg *Message) error {
	var firstError error
	for i := range ms.senders {
		err := ms.retry(ctx, func() error {
			return ms.senders[i].Info(ctx, msg)
		})
		if err != nil {
			ms.logger.Logf("multi-sender: Info %T: %v", ms.senders[i], err)
			if firstError == nil {
				firstError = err
			}
		}
	}
	if firstError == nil && len(ms.senders) > 0 {
		ms.logger.Logf("multi-sender: sent %d info notifications to %v", len(ms.senders), ms.senderTypes())
	}
	return firstError
}

func (ms *MultiSender) Critical(ctx context.Context, msg *Message) error {
	var firstError error
	for i := range ms.senders {
		err := ms.retry(ctx, func() error {
			return ms.senders[i].Critical(ctx, msg)
		})
		if err != nil {
			ms.logger.Logf("multi-sender: Critical %T: %v", ms.senders[i], err)
			if firstError == nil {
				firstError = err
			}
		}
	}
	if firstError == nil && len(ms.senders) > 0 {
		ms.logger.Logf("multi-sender: sent %d critical notifications to %v", len(ms.senders), ms.senderTypes())
	}
	return firstError
}

func (ms *MultiSender) retry(ctx context.Context, f func() error) error {
	if ms.retryConfig != nil {
		backoff, err := setupBackoff(ms.retryConfig)
		if err != nil {
			return fmt.Errorf("retry: %v", err)
		}
		return retry.Do(ctx, backoff, func(ctx context.Context) error {
			return isRetryableError(f())
		})
	}
	return f()
}

func isRetryableError(err error) error {
	if err == nil {
		return nil
	}
	if os.IsTimeout(err) || strings.Contains(err.Error(), "no such host") {
		return retry.RetryableError(err)
	}
	return nil
}
