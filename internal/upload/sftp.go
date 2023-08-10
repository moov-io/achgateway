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
	go_sftp "github.com/moov-io/go-sftp"
)

type SFTPTransferAgent struct {
	client go_sftp.Client
	cfg    service.UploadAgent
	logger log.Logger
}

func newSFTPTransferAgent(logger log.Logger, cfg *service.UploadAgent) (*SFTPTransferAgent, error) {
	if cfg == nil || cfg.SFTP == nil {
		return nil, errors.New("nil SFTP config")
	}

	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), cfg.SFTP.Hostname); err != nil {
		return nil, fmt.Errorf("sftp: %s is not whitelisted: %v", cfg.SFTP.Hostname, err)
	}

	client, err := go_sftp.NewClient(logger, &go_sftp.ClientConfig{
		Hostname: cfg.SFTP.Hostname,
		Username: cfg.SFTP.Username,
		Password: cfg.SFTP.Password,

		ClientPrivateKey: cfg.SFTP.ClientPrivateKey,
		HostPublicKey:    cfg.SFTP.HostPublicKey,

		Timeout:        cfg.SFTP.DialTimeout,
		MaxConnections: cfg.SFTP.MaxConnections(),
		PacketSize:     cfg.SFTP.MaxPacketSize,
	})
	if err != nil {
		return nil, fmt.Errorf("AA: %w", err)
	}
	return &SFTPTransferAgent{
		client: client,
		cfg:    *cfg,
		logger: logger,
	}, nil
}

func (agent *SFTPTransferAgent) ID() string {
	return agent.cfg.ID
}

func (agent *SFTPTransferAgent) Ping() error {
	if agent == nil {
		return errors.New("nil SFTPTransferAgent")
	}

	return agent.client.Ping()
}

func (agent *SFTPTransferAgent) Close() error {
	if agent == nil {
		return nil
	}
	return agent.client.Close()
}

func (agent *SFTPTransferAgent) InboundPath() string {
	return agent.cfg.Paths.Inbound
}

func (agent *SFTPTransferAgent) OutboundPath() string {
	return agent.cfg.Paths.Outbound
}

func (agent *SFTPTransferAgent) ReconciliationPath() string {
	return agent.cfg.Paths.Reconciliation
}

func (agent *SFTPTransferAgent) ReturnPath() string {
	return agent.cfg.Paths.Return
}

func (agent *SFTPTransferAgent) Hostname() string {
	if agent == nil || agent.cfg.SFTP == nil {
		return ""
	}
	return agent.cfg.SFTP.Hostname
}

func (agent *SFTPTransferAgent) Delete(path string) error {
	return agent.client.Delete(path)
}

// uploadFile saves the content of File at the given filename in the OutboundPath directory
//
// The File's contents will always be closed
func (agent *SFTPTransferAgent) UploadFile(f File) error {
	// Take the base of f.Filename and our (out of band) OutboundPath to avoid accepting a write like '../../../../etc/passwd'.
	pathToWrite := filepath.Join(agent.cfg.Paths.Outbound, filepath.Base(f.Filename))

	return agent.client.UploadFile(pathToWrite, f.Contents)
}

func (agent *SFTPTransferAgent) GetInboundFiles() ([]File, error) {
	return agent.readFiles(agent.cfg.Paths.Inbound)
}

func (agent *SFTPTransferAgent) GetReconciliationFiles() ([]File, error) {
	return agent.readFiles(agent.cfg.Paths.Reconciliation)
}

func (agent *SFTPTransferAgent) GetReturnFiles() ([]File, error) {
	return agent.readFiles(agent.cfg.Paths.Return)
}

func (agent *SFTPTransferAgent) readFiles(dir string) ([]File, error) {
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

		reader, err := agent.client.Reader(filenames[i])
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
