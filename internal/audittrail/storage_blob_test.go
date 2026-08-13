// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package audittrail

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/cryptfs"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/stretchr/testify/require"
)

var (
	publicKeyPath = filepath.Join("..", "gpgx", "testdata", "key.pub")
)

func TestBlobStorage(t *testing.T) {
	cfg := &service.AuditTrail{
		BucketURI: "mem://",
		GPG: &service.GPG{
			KeyFile: publicKeyPath,
		},
	}
	store, err := newBlobStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	data := []byte("nacha formatted data")
	if err := store.SaveFile(context.Background(), "ftp.dev.com/saved.ach", data); err != nil {
		t.Fatal(err)
	}

	r, err := store.GetFile(context.Background(), "ftp.dev.com/saved.ach")
	require.NoError(t, err)
	defer r.Close()

	bs, err := io.ReadAll(r)
	require.NoError(t, err)
	if !bytes.Contains(bs, []byte("BEGIN PGP MESSAGE")) {
		t.Errorf("unexpected blob\n%s", string(bs))
	}
}

func TestBlobStorage__AlreadyEncrypted(t *testing.T) {
	cfg := &service.AuditTrail{
		BucketURI: "mem://",
		GPG: &service.GPG{
			KeyFile: publicKeyPath,
		},
	}
	store, err := newBlobStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	plain := []byte("nacha formatted data")
	require.NoError(t, store.SaveFile(context.Background(), "once.ach", plain))

	r, err := store.GetFile(context.Background(), "once.ach")
	require.NoError(t, err)
	wrapped, err := io.ReadAll(r)
	r.Close()
	require.NoError(t, err)
	require.Contains(t, string(wrapped), "BEGIN PGP MESSAGE")

	// Saving a payload already encrypted to the audittrail key must not wrap again.
	require.NoError(t, store.SaveFile(context.Background(), "twice.ach", wrapped))

	r, err = store.GetFile(context.Background(), "twice.ach")
	require.NoError(t, err)
	defer r.Close()

	stored, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, wrapped, stored)
}

func TestBlobStorage__OtherKeyStillEncrypted(t *testing.T) {
	otherKey := writeTempPublicKey(t)
	other, err := cryptfs.FromCryptor(cryptfs.NewGPGEncryptorFile(otherKey))
	require.NoError(t, err)

	foreign, err := other.Disfigure([]byte("nacha formatted data"))
	require.NoError(t, err)
	require.Contains(t, string(foreign), "BEGIN PGP MESSAGE")

	cfg := &service.AuditTrail{
		BucketURI: "mem://",
		GPG: &service.GPG{
			KeyFile: publicKeyPath,
		},
	}
	store, err := newBlobStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.SaveFile(context.Background(), "foreign.ach", foreign))

	r, err := store.GetFile(context.Background(), "foreign.ach")
	require.NoError(t, err)
	defer r.Close()

	stored, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Contains(t, string(stored), "BEGIN PGP MESSAGE")
	require.NotEqual(t, foreign, stored)
}

func TestBlobStorage__NoGPG(t *testing.T) {
	cfg := &service.AuditTrail{
		BucketURI: "mem://",
	}

	store, err := newBlobStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	data := []byte("nacha formatted data")
	err = store.SaveFile(context.Background(), "ftp.dev.com/saved.ach", data)
	require.NoError(t, err)

	r, err := store.GetFile(context.Background(), "ftp.dev.com/saved.ach")
	require.NoError(t, err)
	defer r.Close()

	bs, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, data, bs)
}

func TestBlobStorageErr(t *testing.T) {
	cfg := &service.AuditTrail{
		BucketURI: "bad://",
	}
	if _, err := NewStorage(cfg); err == nil {
		t.Error("expected error")
	}
}

func writeTempPublicKey(t *testing.T) string {
	t.Helper()

	entity, err := openpgp.NewEntity("other", "test", "other@example.com", nil)
	require.NoError(t, err)

	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	require.NoError(t, err)
	require.NoError(t, entity.Serialize(w))
	require.NoError(t, w.Close())

	path := filepath.Join(t.TempDir(), "other.pub")
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0600))
	return path
}
