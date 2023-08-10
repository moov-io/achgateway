// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
	go_ftp "github.com/moov-io/go-ftp"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	ftpAgentUp = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Name: "ftp_agent_up",
		Help: "Status of FTP agent connection ",
	}, []string{"hostname"})
)

// FTPTransferAgent is an FTP implementation of a Agent
type FTPTransferAgent struct {
	client go_ftp.Client
	cfg    service.UploadAgent
	logger log.Logger
}

func newFTPTransferAgent(logger log.Logger, cfg *service.UploadAgent) (*FTPTransferAgent, error) {
	if cfg == nil || cfg.FTP == nil {
		return nil, errors.New("nil FTP config")
	}

	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), cfg.FTP.Hostname); err != nil {
		return nil, fmt.Errorf("ftp: %s is not whitelisted: %v", cfg.FTP.Hostname, err)
	}

	client, err := go_ftp.NewClient(go_ftp.ClientConfig{
		Hostname: cfg.FTP.Hostname,
		Username: cfg.FTP.Username,
		Password: cfg.FTP.Password,

		Timeout:     cfg.FTP.Timeout(),
		DisableEPSV: cfg.FTP.DisableEPSV(),
		CAFile:      cfg.FTP.CAFile(),
	})
	if err != nil {
		return nil, err
	}
	return &FTPTransferAgent{
		client: client,
		cfg:    *cfg,
		logger: logger,
	}, nil
}

func (agent *FTPTransferAgent) ID() string {
	return agent.cfg.ID
}

func (agent *FTPTransferAgent) Ping() error {
	err := agent.client.Ping()
	agent.record(err)
	return err
}

func (agent *FTPTransferAgent) record(err error) {
	if agent == nil {
		return
	}
	if err != nil {
		ftpAgentUp.With("hostname", agent.cfg.FTP.Hostname).Set(0)
	} else {
		ftpAgentUp.With("hostname", agent.cfg.FTP.Hostname).Set(1)
	}
}

func (agent *FTPTransferAgent) Close() error {
	if agent == nil || agent.client == nil {
		return nil
	}
	return agent.client.Close()
}

func (agent *FTPTransferAgent) InboundPath() string {
	return agent.cfg.Paths.Inbound
}

func (agent *FTPTransferAgent) OutboundPath() string {
	return agent.cfg.Paths.Outbound
}

func (agent *FTPTransferAgent) ReconciliationPath() string {
	return agent.cfg.Paths.Reconciliation
}

func (agent *FTPTransferAgent) ReturnPath() string {
	return agent.cfg.Paths.Return
}

func (agent *FTPTransferAgent) Hostname() string {
	if agent == nil || agent.cfg.FTP == nil {
		return ""
	}
	return agent.cfg.FTP.Hostname
}

func (agent *FTPTransferAgent) Delete(path string) error {
	return agent.client.Delete(path)
}

// uploadFile saves the content of File at the given filename in the OutboundPath directory
//
// The File's contents will always be closed
func (agent *FTPTransferAgent) UploadFile(f File) error {
	if agent == nil || agent.cfg.FTP == nil {
		return errors.New("missing FTP client or config")
	}

	pathToWrite := filepath.Join(agent.OutboundPath(), f.Filename)
	return agent.client.UploadFile(pathToWrite, f.Contents)
}

func (agent *FTPTransferAgent) GetInboundFiles() ([]File, error) {
	return agent.readFiles(agent.cfg.Paths.Inbound)
}

func (agent *FTPTransferAgent) GetReconciliationFiles() ([]File, error) {
	return agent.readFiles(agent.cfg.Paths.Reconciliation)
}

func (agent *FTPTransferAgent) GetReturnFiles() ([]File, error) {
	return agent.readFiles(agent.cfg.Paths.Return)
}

func (agent *FTPTransferAgent) readFiles(dir string) ([]File, error) {
	var files []File

	filenames, err := agent.client.ListFiles(dir)
	if err != nil {
		return nil, err
	}

	for i := range filenames {
		// Ignore hidden files
		if strings.HasPrefix(filenames[i], ".") {
			continue
		}

		reader, err := agent.client.Reader(filepath.Join(dir, filenames[i]))
		if err != nil {
			return nil, err
		}
		files = append(files, File{
			Filename: filepath.Base(filenames[i]),
			Contents: reader,
		})
	}

	return files, nil
}
