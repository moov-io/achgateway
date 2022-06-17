package alerting

import (
	"fmt"

	"github.com/moov-io/achgateway/internal/service"
)

type Alerters []Alerter

type Alerter interface {
	AlertError(err error) error
}

type MockAlerter struct{}

func (mn *MockAlerter) AlertError(e error) error {
	return nil
}

func NewAlerters(cfg service.ErrorAlerting) (Alerters, error) {
	var alerters []Alerter
	switch {
	case cfg.Slack != nil:
		alerter, err := NewSlackAlerter(cfg.Slack)
		if err != nil {
			return nil, err
		}
		alerters = append(alerters, alerter)
		fallthrough
	case cfg.PagerDuty != nil:
		alerter, err := NewPagerDutyAlerter(cfg.PagerDuty)
		if err != nil {
			return nil, err
		}
		alerters = append(alerters, alerter)
	}

	if len(alerters) == 0 {
		return []Alerter{&MockAlerter{}}, nil
	}

	return alerters, nil
}

func (s Alerters) AlertError(e error) error {
	if e == nil {
		return nil
	}

	for _, alerter := range s {
		err := alerter.AlertError(e)
		if err != nil {
			return fmt.Errorf("alerting error: %v", err)
		}
	}

	return nil
}
