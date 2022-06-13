package alerting

import (
	"github.com/moov-io/achgateway/internal/service"
)

type Alerter interface {
	AlertError(err error) error
}

type MockAlerter struct{}

func (mn *MockAlerter) AlertError(e error) error {
	return nil
}

func NewAlerters(cfg service.ErrorAlerting) ([]Alerter, error) {
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
