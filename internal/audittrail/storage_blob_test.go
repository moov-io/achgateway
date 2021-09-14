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
	keyPath = filepath.Join("..", "..", "internal", "gpgx", "testdata", "moov.pub")
)

func TestBlobStorage(t *testing.T) {
	cfg := &service.AuditTrail{
		BucketURI: "mem://",
		GPG: &service.GPG{
			KeyFile: keyPath,
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

func TestBlobStorageErr(t *testing.T) {
	cfg := &service.AuditTrail{
		BucketURI: "bad://",
	}
	if _, err := NewStorage(cfg); err == nil {
		t.Error("expected error")
	}
}
