// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package audittrail

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/stretchr/testify/require"
)

var (
	keyPath = filepath.Join("..", "..", "internal", "gpgx", "testdata", "moov.pub")
	ppdPath = filepath.Join("..", "..", "testdata", "ppd-debit.ach")
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

	file, err := ach.ReadFile(ppdPath)
	require.NoError(t, err)

	if err := store.SaveFile("ftp.dev.com", "saved.ach", file); err != nil {
		t.Fatal(err)
	}

	path := fmt.Sprintf("files/ftp.dev.com/%s/saved.ach", time.Now().Format("2006-01-02"))
	r, err := store.GetFile(path)
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
