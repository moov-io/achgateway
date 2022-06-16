// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package audittrail

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
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
	if err := store.SaveFile("ftp.dev.com/saved.ach", data); err != nil {
		t.Fatal(err)
	}

	r, err := store.GetFile("ftp.dev.com/saved.ach")
	require.NoError(t, err)
	defer r.Close()

	bs, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	if !bytes.Contains(bs, []byte("BEGIN PGP MESSAGE")) {
		t.Errorf("unexpected blob\n%s", string(bs))
	}
}

func TestBlobStorage__NoGPG(t *testing.T) {
	cfg := &service.AuditTrail{
		BucketURI: "mem://",
	}

	store, err := newBlobStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	data := []byte("nacha formatted data")
	err = store.SaveFile("ftp.dev.com/saved.ach", data)
	require.NoError(t, err)

	r, err := store.GetFile("ftp.dev.com/saved.ach")
	require.NoError(t, err)
	defer r.Close()

	bs, err := ioutil.ReadAll(r)
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
