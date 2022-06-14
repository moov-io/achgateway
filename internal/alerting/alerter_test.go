package alerting

import (
	"errors"
	"os"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNewAlertersPagerDuty(t *testing.T) {
	if os.Getenv("PD_API_KEY") == "" {
		t.Skip("Skip TestNewAlertersPagerDuty as PD_API_KEY is not set")
	}
	var cfg service.ErrorAlerting
	var alerters []Alerter
	var err error

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

func TestNewAlertersSlack(t *testing.T) {
	if os.Getenv("SLACK_ACCESS_TOKEN") == "" {
		t.Skip("Skip TestNewAlertersSlack as SLACK_ACCESS_TOKEN is not set")
	}
	var cfg service.ErrorAlerting
	var alerters []Alerter
	var err error

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

func TestNewAlertersPagerDutyAndSlack(t *testing.T) {
	if os.Getenv("PD_API_KEY") == "" && os.Getenv("SLACK_ACCESS_TOKEN") == "" {
		t.Skip("Skip as PD_API_KEY and SLACK_ACCESS_TOKEN are not set")
	}
	var cfg service.ErrorAlerting
	var alerters []Alerter
	var err error

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
