// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package transform

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/gpgx"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
	"github.com/moov-io/cryptfs"

	"github.com/ProtonMail/go-crypto/openpgp"
)

type GPGEncryption struct {
	encryptor  *cryptfs.FS
	signingKey openpgp.EntityList
}

func NewGPGEncryptor(logger log.Logger, cfg *service.GPG) (*GPGEncryption, error) {
	if cfg == nil {
		return nil, errors.New("missing GPG config")
	}
	logger = logger.Set("service", log.String("GPG encryption"))

	out := &GPGEncryption{}

	cc, err := cryptfs.FromCryptor(cryptfs.NewGPGEncryptorFile(cfg.KeyFile))
	if err != nil {
		return nil, err
	}
	out.encryptor = cc

	// Read a signing key if it exists
	if cfg.Signer != nil {
		privKey, err := gpgx.ReadPrivateKeyFile(cfg.Signer.KeyFile, []byte(cfg.Signer.Password()))
		if err != nil {
			return nil, err
		}
		out.signingKey = privKey
	}

	return out, nil
}

func (morph *GPGEncryption) Transform(res *Result) (*Result, error) {
	var buf bytes.Buffer
	if err := ach.NewWriter(&buf).Write(res.File); err != nil {
		return res, err
	}

	bs, err := morph.encryptor.Disfigure(buf.Bytes())
	if err != nil {
		return res, err
	}

	// Sign the file after encrypting it
	if len(morph.signingKey) > 0 {
		bs, err = gpgx.Sign(bs, morph.signingKey)
		if err != nil {
			return res, err
		}
	}

	res.Encrypted = bs
	return res, nil
}

func (morph *GPGEncryption) String() string {
	if morph == nil {
		return fmt.Sprintf("GPG: <nil>")
	}
	return fmt.Sprintf("GPG{...}")
}
