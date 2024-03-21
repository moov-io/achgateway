package storage

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

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

func (e *encrypted) Open(path string) (fs.File, error) {
	fd, err := e.underlying.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := fd.Stat()
	if err != nil {
		return nil, err
	}

	defer func() {
		if fd != nil {
			fd.Close()
		}
	}()

	bs, err := io.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	if e.crypt != nil {
		bs, err = e.crypt.Reveal(bs)
		if err != nil {
			return nil, err
		}
	}

	f, ok := fd.(*file)
	if !ok {
		return nil, fmt.Errorf("unexpected file of type %T", fd)
	}

	return &buffer{
		b:        bytes.NewBuffer(bs),
		info:     info,
		filename: f.Filename(),
		fullpath: f.FullPath(),
	}, nil
}

func (e *encrypted) ReadDir(name string) ([]fs.DirEntry, error) {
	return e.underlying.ReadDir(name)
}

var _ fs.ReadDirFS = (&encrypted{})

func (e *encrypted) Glob(pattern string) ([]FileStat, error) {
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

func (e *encrypted) String() string {
	return fmt.Sprintf("storage.encrypted{%#v}", e.underlying)
}
