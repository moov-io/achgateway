// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package sshx

import (
	"bytes"
	"crypto/rsa"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestSSHX__read(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "rsa-2048.pub"))
	require.NoError(t, err)
	key, err := ReadPubKey(data)
	require.NoError(t, err)

	if pk, ok := key.(ssh.CryptoPublicKey); ok {
		t.Logf("ssh: pk=%#v", pk)
		if pk, ok := pk.CryptoPublicKey().(*rsa.PublicKey); ok {
			t.Logf("rsa: pk=%#v", pk)

			var buf bytes.Buffer
			w, err := armor.Encode(&buf, openpgp.PublicKeyType, make(map[string]string))
			if err != nil {
				t.Fatal(err)
			}

			pgpKey := packet.NewRSAPublicKey(time.Now(), pk)
			require.NotNil(t, pgpKey)

			w.Close()
		}
	}
}

func TestSSHX_ReadPubKey(t *testing.T) {
	// TODO(adam): test with '-----BEGIN RSA PRIVATE KEY-----' PKCS#8 format

	check := func(t *testing.T, data []byte) {
		key, err := ReadPubKey(data)
		if key == nil || err != nil {
			t.Fatalf("PublicKey=%v error=%v", key, err)
		}

		// base64 Encoded
		data = []byte(base64.StdEncoding.EncodeToString(data))
		key, err = ReadPubKey(data)
		if key == nil || err != nil {
			t.Fatalf("PublicKey=%v error=%v", key, err)
		}
	}

	// Keys generated with 'ssh-keygen -t rsa -b 2048 -f test' (or 4096)
	data, err := os.ReadFile(filepath.Join("testdata", "rsa-2048.pub"))
	require.NoError(t, err)
	check(t, data)

	data, err = os.ReadFile(filepath.Join("testdata", "rsa-4096.pub"))
	require.NoError(t, err)
	check(t, data)
}
