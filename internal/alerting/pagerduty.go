package alerting

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/moov-io/achgateway/internal/service"
)

type PagerDuty struct {
	client     *pagerduty.Client
	routingKey string
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
	if e == nil {
		return nil
	}

	hostName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("getting host name: %v", err)
	}

	details := make(map[string]string)
	for i := 1; i < 5; i++ {
		if _, file, line, ok := runtime.Caller(i); ok {
			caller := fmt.Sprintf("%s:%d", file, line)
			details[fmt.Sprintf("trace_%d", i)] = caller
		}
	}

	dedupKey := e.Error()
	details["dedupKey"] = dedupKey
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

	if pd.client == nil {
		return errors.New("nil PD client")
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
