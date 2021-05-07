package alerting

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/moov-io/ach-conductor/internal/service"
)

type Alerter interface {
	AlertError(err error) error
}

type PagerDuty struct {
	client     *pagerduty.Client
	routingKey string
}

func NewAlerter(cfg service.ErrorAlerting) (Alerter, error) {
	switch {
	case cfg.PagerDuty != nil:
		return NewPagerDutyAlerter(cfg.PagerDuty)
	case cfg.Mock != nil && cfg.Mock.Enabled:
		return &MockAlerter{}, nil
	}

	return nil, errors.New("no configurations found to create alerter")
}

func NewPagerDutyAlerter(cfg *service.PagerDutyAlerting) (*PagerDuty, error) {
	notifier := &PagerDuty{
		client:     pagerduty.NewClient(cfg.ApiKey),
		routingKey: cfg.RoutingKey,
	}
	if err := notifier.ping(); err != nil {
		return nil, err
	}
	return notifier, nil
}

func (pd *PagerDuty) AlertError(e error) error {
	details := map[string]string{}

	hostName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("getting host name: %v", err)
	}

	dedupKey := e.Error()
	if _, file, line, ok := runtime.Caller(1); ok {
		location := fmt.Sprintf("%s:%d", file, line)
		details["location"] = location
		dedupKey += location
	}

	errorHash := fmt.Sprintf("%x", sha256.Sum256([]byte(dedupKey)))

	event := &pagerduty.V2Event{
		RoutingKey: pd.routingKey,
		Action:     "trigger",
		DedupKey:   errorHash,
		Payload: &pagerduty.V2Payload{
			Summary:   e.Error(),
			Source:    hostName,
			Severity:  "critical",
			Timestamp: time.Now().Format(time.RFC3339),
			Details:   details,
		},
	}

	_, err = pd.client.ManageEvent(event)
	if err != nil {
		return fmt.Errorf("creating event in PagerDuty: %v", err)
	}

	return nil
}

func (pd *PagerDuty) ping() error {
	if pd == nil || pd.client == nil {
		return errors.New("pagerduty: nil")
	}

	// make a call and verify we don't error
	resp, err := pd.client.ListAbilities()
	if err != nil {
		return fmt.Errorf("pagerduty list abilities: %v", err)
	}
	if len(resp.Abilities) <= 0 {
		return fmt.Errorf("pagerduty: missing abilities")
	}

	return nil
}
