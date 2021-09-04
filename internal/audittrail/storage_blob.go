// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package audittrail

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/gpgx"
	"github.com/moov-io/achgateway/internal/output"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/transform"
	"golang.org/x/crypto/openpgp"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"
)

// blobStorage implements Storage with gocloud.dev/blob which allows
// clients to use AWS S3, GCP Storage, and Azure Storage.
type blobStorage struct {
	id              string
	bucket          *blob.Bucket
	outputFormatter *output.NACHA
	pubKey          openpgp.EntityList
}

func newBlobStorage(cfg *service.AuditTrail) (*blobStorage, error) {
	storage := &blobStorage{
		id:              cfg.ID,
		outputFormatter: &output.NACHA{},
	}

	bucket, err := blob.OpenBucket(context.Background(), cfg.BucketURI)
	if err != nil {
		return nil, err
	}
	storage.bucket = bucket

	if cfg.GPG != nil {
		pubKey, err := gpgx.ReadArmoredKeyFile(cfg.GPG.KeyFile)
		if err != nil {
			return nil, err
		}
		storage.pubKey = pubKey
	}

	// set default values for metrics
	uploadFilesErrors.With("type", "blob", "id", cfg.ID).Add(0)
	uploadedFilesCounter.With("type", "blob", "id", cfg.ID).Add(0)

	return storage, nil
}

func (bs *blobStorage) Close() error {
	if bs == nil {
		return nil
	}
	return bs.bucket.Close()
}

func (bs *blobStorage) SaveFile(filepath string, file *ach.File) error {
	result := &transform.Result{File: file}

	var buf bytes.Buffer
	if err := bs.outputFormatter.Format(&buf, result); err != nil {
		uploadFilesErrors.With("type", "blob", "id", bs.id).Add(1)
		return err
	}

	encrypted, err := gpgx.Encrypt(buf.Bytes(), bs.pubKey)
	if err != nil {
		uploadFilesErrors.With("type", "blob", "id", bs.id).Add(1)
		return err
	}

	w, err := bs.bucket.NewWriter(context.Background(), filepath, nil)
	if err != nil {
		uploadFilesErrors.With("type", "blob", "id", bs.id).Add(1)
		return err
	}

	_, copyErr := w.Write(encrypted)
	closeErr := w.Close()

	if copyErr != nil || closeErr != nil {
		uploadFilesErrors.With("type", "blob", "id", bs.id).Add(1)
		return fmt.Errorf("copyErr=%v closeErr=%v", copyErr, closeErr)
	}

	// increment our metrics counter
	uploadedFilesCounter.With("type", "blob", "id", bs.id).Add(1)

	return nil
}

func (bs *blobStorage) GetFile(filepath string) (io.ReadCloser, error) {
	r, err := bs.bucket.NewReader(context.Background(), filepath, nil)
	if err != nil {
		return nil, fmt.Errorf("get file: %v", err)
	}
	return r, nil
}
