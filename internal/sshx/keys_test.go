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
	// TODO(adam): finish up with transform.GPGEncryption

	// 	fd, err := os.Open(filepath.Join("testdata", "rsa-2048.pub"))
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	t.Cleanup(func() { fd.Close() })

	// 	block, err := armor.Decode(fd)
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	if block.Type != openpgp.PublicKeyType {
	// 		t.Fatal(err)
	// 	}

	// 	reader := packet.NewReader(block.Body)
	// 	pkt, err := reader.Next()
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}

	// 	if key, ok := pkt.(*packet.PublicKey); !ok {
	// 		t.Errorf("%T", pkt)
	// 	} else {
	// 		t.Logf("key=%#v", key)
	// 	}

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
			t.Logf("pgpKey=%#v", pgpKey)

			w.Close()

			// to := createEntityFromKeys(pubKey, privKey)

			// w, err := armor.Encode(os.Stdout, "Message", make(map[string]string))
			// kingpin.FatalIfError(err, "Error creating OpenPGP Armor: %s", err)
			// defer w.Close()

			// plain, err := openpgp.Encrypt(w, []*openpgp.Entity{to}, nil, nil, nil)
			// kingpin.FatalIfError(err, "Error creating entity for encryption")
			// defer plain.Close()
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
