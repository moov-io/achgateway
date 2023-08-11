// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"fmt"

	"github.com/moov-io/achgateway/internal/service"

	"github.com/moov-io/base/log"
)

// Agent represents an interface for uploading and retrieving ACH files from a remote service.
type Agent interface {
	ID() string

	GetInboundFiles() ([]File, error)
	GetReconciliationFiles() ([]File, error)
	GetReturnFiles() ([]File, error)
	UploadFile(f File) error
	Delete(path string) error

	InboundPath() string
	OutboundPath() string
	ReconciliationPath() string
	ReturnPath() string
	Hostname() string

	Ping() error
	Close() error
}

func New(logger log.Logger, cfg service.UploadAgents, id string) (Agent, error) {
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
	return agent, nil
}
