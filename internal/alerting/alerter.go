package alerting

import (
	"fmt"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
)

type Alerter interface {
	Alert(err error) error
}

type MockAlerter struct{}

func (mn *MockAlerter) Alert(e error) error {
	return nil
}

type Alerters struct {
	logger log.Logger
	errors []Alerter
}

func (a *Alerters) Error(err error) {
	if a == nil {
		return
	}
	for i := range a.errors {
		if sendErr := a.errors[i].Alert(err); sendErr != nil {
			a.logger.Error().LogErrorf("ERROR sending error: %v", err)
		}
	}
}

func NewAlerters(logger log.Logger, errors service.AlertingConfig) (*Alerters, error) {
	alerters := &Alerters{
		logger: logger,
	}

	var err error

	alerters.errors, err = makeAlerters(errors)
	if err != nil {
		return nil, fmt.Errorf("setting up error alerters: %v", err)
	}

	return alerters, nil
}

func makeAlerters(cfg service.AlertingConfig) ([]Alerter, error) {
	var alerters []Alerter

	if cfg.Slack != nil {
		alerter, err := NewSlackAlerter(cfg.Slack)
		if err != nil {
			return nil, err
		}
		alerters = append(alerters, alerter)
	}

	if cfg.PagerDuty != nil {
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
