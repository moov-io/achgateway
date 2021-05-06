// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package gpgx

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	password = []byte("password")

	privateKeyPath = filepath.Join("testdata", "moov.key")
	publicKeyPath  = filepath.Join("testdata", "moov.pub")
)

func TestGPG(t *testing.T) {
	// Encrypt
	pubKey, err := ReadArmoredKeyFile(publicKeyPath)
	require.NoError(t, err)
	msg, err := Encrypt([]byte("hello, world"), pubKey)
	require.NoError(t, err)
	if len(msg) == 0 {
		t.Error("empty encrypted message")
	}

	// Decrypt
	privKey, err := ReadPrivateKeyFile(privateKeyPath, password)
	require.NoError(t, err)
	require.NoError(t, err)
	out, err := Decrypt(msg, privKey)
	require.NoError(t, err)

	if v := string(out); v != "hello, world" {
		t.Errorf("got %q", v)
	}
}
