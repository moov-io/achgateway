// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package audittrail

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/moov-io/achgateway/internal/gpgx"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/telemetry"
	"github.com/moov-io/cryptfs"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

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
	id      string
	bucket  *blob.Bucket
	cryptor *cryptfs.FS
	gpgKeys openpgp.EntityList
}

func newBlobStorage(cfg *service.AuditTrail) (*blobStorage, error) {
	storage := &blobStorage{
		id: cfg.ID,
	}

	bucket, err := blob.OpenBucket(context.Background(), cfg.BucketURI)
	if err != nil {
		return nil, err
	}
	storage.bucket = bucket

	if cfg.GPG != nil {
		storage.cryptor, err = cryptfs.FromCryptor(cryptfs.NewGPGEncryptorFile(cfg.GPG.KeyFile))
		if err != nil {
			return nil, err
		}
		storage.gpgKeys, err = gpgx.ReadArmoredKeyFile(cfg.GPG.KeyFile)
		if err != nil {
			return nil, err
		}
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

func (bs *blobStorage) SaveFile(ctx context.Context, path string, data []byte) error {
	ctx, span := telemetry.StartSpan(ctx, "audittrail-save-file", trace.WithAttributes(
		attribute.String("achgateway.path", path),
		attribute.Int("achgateway.data_bytes", len(data)),
	))
	defer span.End()

	var encrypted []byte
	var err error
	// Always encrypt with the configured audittrail key, unless the
	// payload is already encrypted to that same key.
	if bs.cryptor != nil && !alreadyEncryptedTo(data, bs.gpgKeys) {
		encrypted, err = bs.cryptor.Disfigure(data)
	} else {
		encrypted = data
	}
	if err != nil {
		uploadFilesErrors.With("type", "blob", "id", bs.id).Add(1)
		return err
	}

	exists, err := bs.bucket.Exists(ctx, path)
	if exists {
		return nil
	}
	if err != nil {
		uploadFilesErrors.With("type", "blob", "id", bs.id).Add(1)
		return err
	}

	w, err := bs.bucket.NewWriter(ctx, path, nil)
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

func (bs *blobStorage) GetFile(ctx context.Context, path string) (io.ReadCloser, error) {
	ctx, span := telemetry.StartSpan(ctx, "audittrail-get-file", trace.WithAttributes(
		attribute.String("achgateway.path", path),
	))
	defer span.End()

	r, err := bs.bucket.NewReader(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get file: %v", err)
	}
	return r, nil
}

func alreadyEncryptedTo(data []byte, keys openpgp.EntityList) bool {
	if len(keys) == 0 {
		return false
	}
	recipients, err := encryptedToKeyIDs(data)
	if err != nil || len(recipients) == 0 {
		return false
	}
	ours := keyIDs(keys)
	for _, recipient := range recipients {
		if recipient == 0 {
			continue
		}
		for _, oursID := range ours {
			if recipient == oursID {
				return true
			}
		}
	}
	return false
}

func encryptedToKeyIDs(data []byte) ([]uint64, error) {
	body := bytes.TrimSpace(data)
	if !bytes.HasPrefix(body, []byte("-----BEGIN PGP ")) {
		return nil, nil
	}
	block, err := armor.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	packets := packet.NewReader(block.Body)
	var ids []uint64
	for {
		p, err := packets.Next()
		if err != nil {
			if err == io.EOF {
				return ids, nil
			}
			return ids, err
		}
		switch p := p.(type) {
		case *packet.EncryptedKey:
			ids = append(ids, p.KeyId)
		case *packet.SymmetricallyEncrypted, *packet.AEADEncrypted:
			return ids, nil
		}
	}
}

func keyIDs(keys openpgp.EntityList) []uint64 {
	var ids []uint64
	for _, entity := range keys {
		if entity.PrimaryKey != nil {
			ids = append(ids, entity.PrimaryKey.KeyId)
		}
		for i := range entity.Subkeys {
			if entity.Subkeys[i].PublicKey != nil {
				ids = append(ids, entity.Subkeys[i].PublicKey.KeyId)
			}
		}
	}
	return ids
}
