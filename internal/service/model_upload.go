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
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/moov-io/achgateway/internal/mask"
)

type UploadAgents struct {
	Agents         []UploadAgent
	Retry          *UploadRetry
	DefaultAgentID string
}

func (ua UploadAgents) Find(id string) *UploadAgent {
	for i := range ua.Agents {
		if ua.Agents[i].ID == id {
			return &ua.Agents[i]
		}
	}
	return nil
}

func (ua UploadAgents) Validate() error {
	if err := ua.Retry.Validate(); err != nil {
		return fmt.Errorf("retry: %v", err)
	}
	return nil
}

type UploadAgent struct {
	ID            string
	FTP           *FTP
	SFTP          *SFTP
	Mock          *MockAgent
	Paths         UploadPaths
	Notifications *UploadNotifiers

	// AllowedIPs is a comma separated list of IP addresses and CIDR ranges
	// where connections are allowed. If this value is non-empty remote servers
	// not within these ranges will not be connected to.
	AllowedIPs string
}

func (cfg *UploadAgent) SplitAllowedIPs() []string {
	if cfg.AllowedIPs != "" {
		return strings.Split(cfg.AllowedIPs, ",")
	}
	return nil
}

type FTP struct {
	Hostname string
	Username string
	Password string

	CAFilepath   string
	DialTimeout  time.Duration
	DisabledEPSV bool
}

func (cfg *FTP) CAFile() string {
	if cfg == nil {
		return ""
	}
	return cfg.CAFilepath
}

func (cfg *FTP) Timeout() time.Duration {
	if cfg == nil || cfg.DialTimeout == 0*time.Second {
		return 10 * time.Second
	}
	return cfg.DialTimeout
}

func (cfg *FTP) DisableEPSV() bool {
	if cfg == nil {
		return false
	}
	return cfg.DisabledEPSV
}

func (cfg *FTP) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("FTP{Hostname=%s, ", cfg.Hostname))
	buf.WriteString(fmt.Sprintf("Username=%s, ", cfg.Username))
	buf.WriteString(fmt.Sprintf("Password=%s}", mask.Password(cfg.Password)))
	return buf.String()
}

type SFTP struct {
	Hostname string
	Username string

	Password         string
	ClientPrivateKey string
	HostPublicKey    string

	DialTimeout           time.Duration
	MaxConnectionsPerFile int
	MaxPacketSize         int
}

func (cfg *SFTP) Timeout() time.Duration {
	if cfg == nil || cfg.DialTimeout == 0*time.Second {
		return 10 * time.Second
	}
	return cfg.DialTimeout
}

func (cfg *SFTP) MaxConnections() int {
	if cfg == nil || cfg.MaxConnectionsPerFile == 0 {
		return 1 // pkg/sftp's default is 64
	}
	return cfg.MaxConnectionsPerFile
}

func (cfg *SFTP) PacketSize() int {
	if cfg == nil || cfg.MaxPacketSize == 0 {
		return 20480
	}
	return cfg.MaxPacketSize
}

func (cfg *SFTP) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("SFTP{Hostname=%s, ", cfg.Hostname))
	buf.WriteString(fmt.Sprintf("Username=%s, ", cfg.Username))
	buf.WriteString(fmt.Sprintf("Password=%s, ", mask.Password(cfg.Password)))
	buf.WriteString(fmt.Sprintf("ClientPrivateKey:%v, ", cfg.ClientPrivateKey != ""))
	buf.WriteString(fmt.Sprintf("HostPublicKey:%v}, ", cfg.HostPublicKey != ""))
	return buf.String()
}

type MockAgent struct{}

type UploadPaths struct {
	Inbound        string
	Outbound       string
	Reconciliation string
	Return         string
}

type UploadNotifiers struct {
	Email     []string
	PagerDuty []string
	Slack     []string
}

type UploadRetry struct {
	Interval   time.Duration
	MaxRetries uint64
}

func (cfg *UploadRetry) Validate() error {
	if cfg == nil {
		return nil
	}
	if cfg.Interval <= 0*time.Second {
		return fmt.Errorf("unexpected %d interval", cfg.Interval)
	}
	if cfg.MaxRetries <= 0 {
		return fmt.Errorf("unexpected %d max retries", cfg.MaxRetries)
	}
	return nil
}
