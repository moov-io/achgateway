// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
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

	require.NoError(t, sender.Info(msg))
	require.NoError(t, sender.Critical(msg))

	sender.senders = append(sender.senders, &MockSender{})

	require.NoError(t, sender.Info(msg))
	require.NoError(t, sender.Critical(msg))
}

func TestMultiSenderErr(t *testing.T) {
	sendErr := errors.New("bad error")

	sender := &MultiSender{
		logger: log.NewTestLogger(),
		senders: []Sender{
			&MockSender{Err: sendErr},
		},
	}

	msg := &Message{Direction: Upload}

	require.Equal(t, sender.Info(msg), sendErr)
	require.Equal(t, sender.Critical(msg), sendErr)
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
