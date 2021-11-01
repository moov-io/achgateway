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
	"strings"
	"text/template"
	"time"
)

var (
	DefaultEmailTemplate = template.Must(template.New("email").Parse(`
A file has been {{ .Verb }}ed{{ if .Hostname }}{{ if eq .Verb "upload" }} to{{ else }} from{{end}} {{ .Hostname }}{{end}} - {{ .Filename }}
Name: {{ .CompanyName }}
Debits:  ${{ .DebitTotal }}
Credits: ${{ .CreditTotal }}

Batches: {{ .BatchCount }}
Total Entries: {{ .EntryCount }}
`))
)

type Notifications struct {
	Email     []Email
	PagerDuty []PagerDuty
	Slack     []Slack
	Retry     *NotificationRetries
}

type NotificationRetries struct {
	Interval   time.Duration
	MaxRetries uint64
}

func (cfg Notifications) FindEmails(ids []string) []Email {
	var out []Email
	for i := range ids {
		for j := range cfg.Email {
			if strings.EqualFold(ids[i], cfg.Email[j].ID) {
				out = append(out, cfg.Email[j])
			}
		}
	}
	return out
}

func (cfg Notifications) FindPagerDutys(ids []string) []PagerDuty {
	var out []PagerDuty
	for i := range ids {
		for j := range cfg.PagerDuty {
			if strings.EqualFold(ids[i], cfg.PagerDuty[j].ID) {
				out = append(out, cfg.PagerDuty[j])
			}
		}
	}
	return out
}

func (cfg Notifications) FindSlacks(ids []string) []Slack {
	var out []Slack
	for i := range ids {
		for j := range cfg.Slack {
			if strings.EqualFold(ids[i], cfg.Slack[j].ID) {
				out = append(out, cfg.Slack[j])
			}
		}
	}
	return out
}

func (cfg Notifications) Validate() error {
	for i := range cfg.Email {
		e := cfg.Email[i]
		if e.From == "" || len(e.To) == 0 || e.ConnectionURI == "" || e.CompanyName == "" {
			return errors.New("email: missing configs")
		}
	}
	for i := range cfg.PagerDuty {
		if err := cfg.PagerDuty[i].Validate(); err != nil {
			return err
		}
	}
	for i := range cfg.Slack {
		if err := cfg.Slack[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

type Email struct {
	ID string

	From string
	To   []string

	// ConnectionURI is a URI used to connect with a remote SFTP server.
	// This config typically needs to contain enough values to successfully
	// authenticate with the server.
	// - insecure_skip_verify is an optional parameter for disabling certificate verification
	//
	// Example: smtps://user:pass@localhost:1025/?insecure_skip_verify=true
	ConnectionURI string

	Template    string
	CompanyName string
}

func (cfg Email) Tmpl() *template.Template {
	if cfg.Template == "" {
		return DefaultEmailTemplate
	}
	return template.Must(template.New("custom-email").Parse(cfg.Template))
}

type PagerDuty struct {
	ID         string
	ApiKey     string
	From       string
	ServiceKey string
}

func (cfg PagerDuty) Validate() error {
	if cfg.ID == "" {
		return errors.New("pagerduty: missing id")
	}
	if cfg.ApiKey == "" {
		return errors.New("pagerduty: missing apiKey")
	}
	return nil
}

type Slack struct {
	ID string

	WebhookURL string
}

func (cfg Slack) Validate() error {
	if cfg.ID == "" {
		return errors.New("slack: missing id")
	}
	if cfg.WebhookURL == "" {
		return errors.New("slack: missing webhook url")
	}
	return nil
}
