package storage

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/moov-io/cryptfs"
)

type Chest interface {
	Open(path string) (File, error)
	Glob(pattern string) ([]FileStat, error)

	ReplaceFile(oldpath, newpath string) error
	ReplaceDir(oldpath, newpath string) error

	MkdirAll(path string) error
	RmdirAll(path string) error

	WriteFile(path string, contents []byte) error
}

// New returns a Chest given the configuration provided. It can also wrap that Chest
// in an encryption routine on each operation.
func New(cfg Config) (Chest, error) {
	underlying, err := NewFilesystem(cfg.Filesystem.Directory)
	if err != nil {
		return nil, fmt.Errorf("error initializing filesystem storage: %w", err)
	}

	if cfg.Encryption.AES != nil {
		enc, err := createBase64AESCryptor(cfg.Encryption.AES.Base64Key)
		if err != nil {
			return nil, fmt.Errorf("error creating AES cryptor: %w", err)
		}

		fs, err := cryptfs.New(enc)
		if err != nil {
			return nil, fmt.Errorf("error creating cryptfs: %w", err)
		}

		switch strings.ToLower(cfg.Encryption.Encoding) {
		case "base64":
			fs.SetCoder(cryptfs.Base64())
		}

		return NewEncrypted(underlying, fs), nil
	}

	return underlying, nil
}

func createBase64AESCryptor(key string) (*cryptfs.AESCryptor, error) {
	decoded, err := base64.RawStdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	return cryptfs.NewAESCryptor(decoded)
}
