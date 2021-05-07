package alerting

import (
	"errors"
	"os"
	"testing"

	"github.com/moov-io/ach-conductor/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPagerDutyErrorAlert(t *testing.T) {
	if os.Getenv("PD_API_KEY") == "" {
		t.Skip("Skip PagerDuty notification as PD_API_KEY and PD_ROUTING_KEY are not set")
	}

	cfg := &service.PagerDutyAlerting{
		ApiKey:     os.Getenv("PD_API_KEY"),
		RoutingKey: os.Getenv("PD_ROUTING_KEY"),
	}

	notifier, err := NewPagerDutyAlerter(cfg)
	require.NoError(t, err)
	require.NotNil(t, notifier)

	err = notifier.AlertError(errors.New("error message"))
	require.NoError(t, err)
}
