package storage

import (
	"bytes"
	"io/ioutil"

	"github.com/moov-io/cryptfs"
)

type encrypted struct {
	crypt      *cryptfs.FS
	underlying Chest
}

func NewEncrypted(underlying Chest, crypt *cryptfs.FS) Chest {
	return &encrypted{
		underlying: underlying,
		crypt:      crypt,
	}
}

func (e *encrypted) Open(path string) (File, error) {
	file, err := e.underlying.Open(path)
	if err != nil {
		return nil, err
	}
	bs, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if e.crypt != nil {
		bs, err = e.crypt.Reveal(bs)
		if err != nil {
			return nil, err
		}
	}
	return &buffer{
		b:        bytes.NewBuffer(bs),
		filename: file.Filename(),
		fullpath: file.FullPath(),
	}, nil
}

func (e *encrypted) Glob(pattern string) ([]string, error) {
	return e.underlying.Glob(pattern)
}

func (e *encrypted) ReplaceFile(oldpath, newpath string) error {
	return e.underlying.ReplaceFile(oldpath, newpath)
}

func (e *encrypted) ReplaceDir(oldpath, newpath string) error {
	return e.underlying.ReplaceDir(oldpath, newpath)
}

func (e *encrypted) MkdirAll(path string) error {
	return e.underlying.MkdirAll(path)
}

func (e *encrypted) RmdirAll(path string) error {
	return e.underlying.RmdirAll(path)
}

func (e *encrypted) WriteFile(path string, contents []byte) error {
	var err error
	if e.crypt != nil {
		contents, err = e.crypt.Disfigure(contents)
		if err != nil {
			return err
		}
	}
	return e.underlying.WriteFile(path, contents)
}
