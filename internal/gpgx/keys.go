// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package gpgx

import (
	"bytes"
	"errors"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// ReadArmoredKeyFile attempts to read the filepath and parses an armored GPG key
func ReadArmoredKeyFile(path string) (openpgp.EntityList, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return openpgp.ReadArmoredKeyRing(bytes.NewBuffer(bs))
}

// ReadPrivateKeyFile attempts to read the filepath and parses an armored GPG private key
func ReadPrivateKeyFile(path string, password []byte) (openpgp.EntityList, error) {
	// Read the private key
	entityList, err := ReadArmoredKeyFile(path)
	if err != nil {
		return nil, err
	}
	if len(entityList) == 0 {
		return nil, errors.New("gpg: no entities found")
	}

	entity := entityList[0]

	// Get the passphrase and read the private key.
	if entity.PrivateKey != nil && len(password) > 0 {
		entity.PrivateKey.Decrypt(password)
	}
	for _, subkey := range entity.Subkeys {
		if subkey.PrivateKey != nil && len(password) > 0 {
			subkey.PrivateKey.Decrypt(password)
		}
	}

	return entityList, nil
}
func Sign(message []byte, pubKey openpgp.EntityList) ([]byte, error) {
	if len(pubKey) == 0 {
		return nil, errors.New("sign: missing Entity")
	}

	var out bytes.Buffer
	r := bytes.NewReader(message)
	if err := openpgp.ArmoredDetachSign(&out, pubKey[0], r, nil); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
