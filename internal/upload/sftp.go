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

		SkipDirectoryCreation: cfg.SFTP.SkipDirectoryCreation,
	})
	if err != nil {
		return nil, err
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
	// Take the base of f.Filepath and our (out of band) OutboundPath to avoid accepting a write like '../../../../etc/passwd'.
	pathToWrite := filepath.Join(agent.OutboundPath(), filepath.Base(f.Filepath))

	return agent.client.UploadFile(pathToWrite, f.Contents)
}

func (agent *SFTPTransferAgent) ReadFile(path string) (*File, error) {
	file, err := agent.client.Reader(path)
	if err != nil {
		return nil, fmt.Errorf("sftp open %s failed: %w", path, err)
	}
	return &File{
		Filepath: filepath.Base(file.Filename),
		Contents: file.Contents,
	}, nil
}

func (agent *SFTPTransferAgent) GetInboundFiles() ([]string, error) {
	return agent.readFilepaths(agent.cfg.Paths.Inbound)
}

func (agent *SFTPTransferAgent) GetReconciliationFiles() ([]string, error) {
	return agent.readFilepaths(agent.cfg.Paths.Reconciliation)
}

func (agent *SFTPTransferAgent) GetReturnFiles() ([]string, error) {
	return agent.readFilepaths(agent.cfg.Paths.Return)
}

func (agent *SFTPTransferAgent) readFilepaths(dir string) ([]string, error) {
	filepaths, err := agent.client.ListFiles(dir)
	if err != nil {
		return nil, err
	}

	// remove hidden files from resulting filepaths
	for i := len(filepaths) - 1; i >= 0; i-- {
		if strings.HasPrefix(filepath.Base(filepaths[i]), ".") {
			filepaths = append(filepaths[:i], filepaths[i+1:]...)
		}
	}
	return filepaths, nil
}
