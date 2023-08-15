// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"fmt"
	"sync"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/log"
)

// Agent represents an interface for uploading and retrieving ACH files from a remote service.
type Agent interface {
	ID() string

	GetInboundFiles() ([]string, error)
	GetReconciliationFiles() ([]string, error)
	GetReturnFiles() ([]string, error)
	UploadFile(f File) error
	Delete(path string) error
	ReadFile(path string) (*File, error)

	InboundPath() string
	OutboundPath() string
	ReconciliationPath() string
	ReturnPath() string
	Hostname() string

	Ping() error
	Close() error
}

var (
	createdAgents = &CreatedAgents{}
)

func New(logger log.Logger, cfg service.UploadAgents, id string) (Agent, error) {
	createdAgents.mu.Lock()
	defer createdAgents.mu.Unlock()

	// lookup cached
	for i := range createdAgents.agents {
		if createdAgents.agents[i].ID() == id {
			return createdAgents.agents[i], nil
		}
	}

	// Create the new agent
	var agent Agent
	if conf := cfg.Find(id); conf != nil {
		if conf.FTP != nil {
			aa, err := newFTPTransferAgent(logger, conf)
			if err != nil {
				return nil, err
			}
			agent = aa
		}
		if conf.SFTP != nil {
			aa, err := newSFTPTransferAgent(logger, conf)
			if err != nil {
				return nil, err
			}
			agent = aa
		}
		if conf.Mock != nil {
			agent = &MockAgent{}
		}
	}
	if agent == nil {
		return nil, fmt.Errorf("upload: unknown Agent ID=%s", id)
	}
	if cfg.Retry != nil {
		retr, err := newRetryAgent(logger, agent, cfg.Retry)
		if err != nil {
			return nil, err
		}
		agent = retr
	}
	createdAgents.register(agent)
	return agent, nil
}

type CreatedAgents struct {
	mu          sync.Mutex
	agents      []Agent
	adminServer *admin.Server
}

func RegisterAdminServer(svc *admin.Server) {
	createdAgents.mu.Lock()
	defer createdAgents.mu.Unlock()

	if createdAgents.adminServer == nil && svc != nil {
		createdAgents.adminServer = svc
	}
}

func (as *CreatedAgents) register(agent Agent) {
	// track agent
	as.agents = append(as.agents, agent)

	// register liveness probe
	if as.adminServer != nil {
		kind := fmt.Sprintf("%T", agent)
		as.adminServer.AddLivenessCheck(kind, agent.Ping)
	}
}
