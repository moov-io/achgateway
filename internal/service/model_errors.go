// Licensed to The Moov Authors under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. The Moov Authors licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package service

import (
	"errors"
	"fmt"
)

type ErrorAlerting struct {
	PagerDuty *PagerDutyAlerting
}

func (n ErrorAlerting) Validate() error {
	if n.PagerDuty != nil {
		if err := n.PagerDuty.Validate(); err != nil {
			return fmt.Errorf("pager duty config: %v", err)
		}
	}
	return nil
}

type PagerDutyAlerting struct {
	ApiKey string

	// To send an alert event we need to provide the value of
	// the Integration Key (add API integration to service in PD to get it)
	// as RoutingKey
	RoutingKey string
}

func (cfg PagerDutyAlerting) Validate() error {
	if cfg.ApiKey == "" {
		return errors.New("pagerduty error alerting: apiKey is missing")
	}
	if cfg.RoutingKey == "" {
		return errors.New("pagerduty error alerting: routingKey is missing")
	}
	return nil
}
