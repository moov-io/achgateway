// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/telemetry"
	go_ftp "github.com/moov-io/go-ftp"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

func (agent *FTPTransferAgent) Delete(ctx context.Context, path string) error {
	_, span := telemetry.StartSpan(ctx, "agent-ftp-delete", trace.WithAttributes(
		attribute.String("path", path),
	))
	defer span.End()

	return agent.client.Delete(path)
}

// uploadFile saves the content of File at the given filename in the OutboundPath directory
//
// The File's contents will always be closed
func (agent *FTPTransferAgent) UploadFile(ctx context.Context, f File) error {
	if agent == nil || agent.cfg.FTP == nil {
		return errors.New("missing FTP client or config")
	}

	pathToWrite := filepath.Join(agent.OutboundPath(), f.Filepath)

	_, span := telemetry.StartSpan(ctx, "agent-ftp-upload", trace.WithAttributes(
		attribute.String("path", pathToWrite),
	))
	defer span.End()

	return agent.client.UploadFile(pathToWrite, f.Contents)
}

func (agent *FTPTransferAgent) ReadFile(ctx context.Context, path string) (*File, error) {
	_, span := telemetry.StartSpan(ctx, "agent-ftp-read", trace.WithAttributes(
		attribute.String("path", path),
	))
	defer span.End()

	file, err := agent.client.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ftp open %s failed: %w", path, err)
	}
	return &File{
		Filepath: filepath.Base(file.Filename),
		Contents: file.Contents,
	}, nil
}

func (agent *FTPTransferAgent) GetInboundFiles(ctx context.Context) ([]string, error) {
	return agent.readFilepaths(ctx, agent.cfg.Paths.Inbound)
}

func (agent *FTPTransferAgent) GetReconciliationFiles(ctx context.Context) ([]string, error) {
	return agent.readFilepaths(ctx, agent.cfg.Paths.Reconciliation)
}

func (agent *FTPTransferAgent) GetReturnFiles(ctx context.Context) ([]string, error) {
	return agent.readFilepaths(ctx, agent.cfg.Paths.Return)
}

func (agent *FTPTransferAgent) readFilepaths(ctx context.Context, dir string) ([]string, error) {
	_, span := telemetry.StartSpan(ctx, "agent-ftp-list", trace.WithAttributes(
		attribute.String("path", dir),
	))
	defer span.End()

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
