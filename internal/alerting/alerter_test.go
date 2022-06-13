package alerting

import (
	"errors"
	"os"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewAlerters(t *testing.T) {
	if os.Getenv("PD_API_KEY") == "" && os.Getenv("SLACK_ACCESS_TOKEN") == "" {
		t.Skip("Skip TestNewAlerters as PD_API_KEY and SLACK_ACCESS_TOKEN are not set")
	}
	var cfg service.ErrorAlerting
	var alerters []Alerter
	var err error

	if os.Getenv("PD_API_KEY") != "" {
		cfg = service.ErrorAlerting{
			PagerDuty: &service.PagerDutyAlerting{
				ApiKey:     os.Getenv("PD_API_KEY"),
				RoutingKey: os.Getenv("PD_ROUTING_KEY"),
			},
		}

		alerters, err = NewAlerters(cfg)
		require.NoError(t, err)
		require.Len(t, alerters, 1)
	}

	if os.Getenv("SLACK_ACCESS_TOKEN") != "" {
		cfg = service.ErrorAlerting{
			Slack: &service.SlackAlerting{
				AccessToken: os.Getenv("SLACK_ACCESS_TOKEN"),
				ChannelID:   os.Getenv("SLACK_CHANNEL_ID"),
			},
		}

		alerters, err = NewAlerters(cfg)
		require.NoError(t, err)
		require.Len(t, alerters, 1)
	}

	if os.Getenv("PD_API_KEY") != "" && os.Getenv("SLACK_ACCESS_TOKEN") != "" {
		cfg = service.ErrorAlerting{
			PagerDuty: &service.PagerDutyAlerting{
				ApiKey:     os.Getenv("PD_API_KEY"),
				RoutingKey: os.Getenv("PD_ROUTING_KEY"),
			},
			Slack: &service.SlackAlerting{
				AccessToken: os.Getenv("SLACK_ACCESS_TOKEN"),
				ChannelID:   os.Getenv("SLACK_CHANNEL_ID"),
			},
		}

		alerters, err = NewAlerters(cfg)
		require.NoError(t, err)
		require.Len(t, alerters, 2)

		for _, alerter := range alerters {
			err = alerter.AlertError(errors.New("error message"))
			require.NoError(t, err)
		}
	}
}
