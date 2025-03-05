// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestMultiSender(t *testing.T) {
	logger := log.NewTestLogger()
	cfg := &service.Notifications{}
	notifiers := &service.UploadNotifiers{}

	sender, err := NewMultiSender(logger, cfg, notifiers)
	require.NoError(t, err)

	msg := &Message{Direction: Upload}

	ctx := context.Background()
	require.NoError(t, sender.Info(ctx, msg))
	require.NoError(t, sender.Critical(ctx, msg))

	sender.senders = append(sender.senders, &MockSender{})

	require.NoError(t, sender.Info(ctx, msg))
	require.NoError(t, sender.Critical(ctx, msg))
}

func TestMultiSender_senderTypes(t *testing.T) {
	logger := log.NewTestLogger()
	cfg := &service.Notifications{
		Email: []service.Email{
			{
				ID:   "testing",
				From: "user:pass@localhost:4133",
			},
		},
	}
	notifiers := &service.UploadNotifiers{
		Email: []string{"testing"},
	}

	sender, err := NewMultiSender(logger, cfg, notifiers)
	require.NoError(t, err)

	require.Equal(t, "*notify.Email", sender.senderTypes()) // no password leaked
}

func TestMultiSenderErr(t *testing.T) {
	sendErr := errors.New("bad error")

	sender := &MultiSender{
		logger: log.NewTestLogger(),
		senders: []Sender{
			&MockSender{Err: sendErr},
		},
	}

	ctx := context.Background()
	msg := &Message{Direction: Upload}

	require.Equal(t, sender.Info(ctx, msg), sendErr)
	require.Equal(t, sender.Critical(ctx, msg), sendErr)
}

func TestMulti__Retry(t *testing.T) {
	cfg := &service.Notifications{
		Retry: &service.NotificationRetries{
			Interval:   1 * time.Second,
			MaxRetries: 3,
		},
	}
	ms, err := NewMultiSender(log.NewTestLogger(), cfg, &service.UploadNotifiers{})
	require.NoError(t, err)
	require.NotNil(t, ms.retryConfig)
}
