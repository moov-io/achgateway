// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package transform

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/cryptfs"

	"github.com/stretchr/testify/require"
)

var (
	password = []byte("password")

	pubKeyFile  = filepath.Join("..", "gpgx", "testdata", "key.pub")
	privKeyFile = filepath.Join("..", "gpgx", "testdata", "key.priv")
)

func TestGPGEncryptor(t *testing.T) {
	cfg := &service.GPG{
		KeyFile: pubKeyFile,
	}
	gpg, err := NewGPGEncryptor(cfg)
	require.NoError(t, err)

	// Read file and encrypt it
	orig, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)
	res, err := gpg.Transform(&Result{File: orig})
	require.NoError(t, err)

	dd, err := cryptfs.FromCryptor(cryptfs.NewGPGDecryptorFile(privKeyFile, password))
	require.NoError(t, err)
	decrypted, err := dd.Reveal(res.Encrypted)
	require.NoError(t, err)

	if err := compareKeys(orig, decrypted); err != nil {
		t.Error(err)
	}
}

func TestGPGAndSign(t *testing.T) {
	cfg := &service.GPG{
		KeyFile: pubKeyFile,
		Signer: &service.Signer{
			KeyFile:     privKeyFile,
			KeyPassword: "password",
		},
	}
	gpg, err := NewGPGEncryptor(cfg)
	require.NoError(t, err)

	// Read file and encrypt it
	orig, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)
	res, err := gpg.Transform(&Result{File: orig})
	require.NoError(t, err)

	if len(res.Encrypted) == 0 {
		t.Errorf("got no encrypted bytes")
	}
}

func compareKeys(orig *ach.File, decrypted []byte) error {
	if orig == nil {
		return errors.New("missing Original")
	}
	if len(decrypted) == 0 {
		return errors.New("missing decrypted File")
	}

	// marshal the original to bytes so we can compare
	var origBuf bytes.Buffer
	if err := ach.NewWriter(&origBuf).Write(orig); err != nil {
		return err
	}
	origBS := origBuf.Bytes()

	// byte-by-byte compare
	if len(origBS) != len(decrypted) {
		return fmt.Errorf("orig=%d decrypted=%d", len(origBS), len(decrypted))
	}
	for i := range origBS {
		if origBS[i] != decrypted[i] {
			return fmt.Errorf("byte #%d '%v' vs '%v'", i, origBS[i], decrypted[i])
		}
	}

	return nil
}
