package alerting

import (
	"errors"
	"os"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSlackErrorAlert(t *testing.T) {
	if os.Getenv("SLACK_ACCESS_TOKEN") == "" {
		t.Skip("Skip Slack notification as SLACK_ACCESS_TOKEN and SLACK_CHANNEL_ID are not set")
	}

	cfg := &service.SlackAlerting{
		AccessToken: os.Getenv("SLACK_ACCESS_TOKEN"),
		ChannelID:   os.Getenv("SLACK_CHANNEL_ID"),
	}

	notifier, err := NewSlackAlerter(cfg)
	require.NoError(t, err)
	require.NotNil(t, notifier)

	err = notifier.AlertError(errors.New("error message"))
	require.NoError(t, err)
}
