// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package audittrail

import (
	"errors"
	"io"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/service"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	uploadedFilesCounter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "audittrail_uploaded_files",
		Help: "Counter of ACH files uploaded to audit trail storage",
	}, []string{"type"})

	uploadFilesErrors = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "audittrail_upload_errors",
		Help: "Counter of errors encountered when attempting ACH files upload",
	}, []string{"type"})
)

// Storage is an interface for saving and encrypting ACH files for
// records retention. This is often a requirement of agreements.
//
// File retention after upload is not part of this storage.
type Storage interface {
	// SaveFile will encrypt and copy the ACH file to the configured file storage.
	SaveFile(filename string, file *ach.File) error

	GetFile(filepath string) (io.ReadCloser, error)

	Close() error
}

func NewStorage(cfg *service.AuditTrail) (Storage, error) {
	if cfg == nil {
		return newMockStorage(), nil
	}
	if cfg.BucketURI != "" {
		return newBlobStorage(cfg)
	}
	return nil, errors.New("unknown storage config")
}
