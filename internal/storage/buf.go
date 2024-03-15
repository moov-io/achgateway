package storage

import (
	"bytes"
	"io/fs"
)

type buffer struct {
	b *bytes.Buffer

	info fs.FileInfo

	filename string
	fullpath string
}

func (b *buffer) Filename() string {
	return b.filename
}

func (b *buffer) FullPath() string {
	return b.fullpath
}

func (b *buffer) Stat() (fs.FileInfo, error) {
	return b.info, nil
}

func (b *buffer) Read(data []byte) (int, error) {
	return b.b.Read(data)
}

func (b *buffer) Close() error {
	b.b.Reset()
	return nil
}

var _ File = (&buffer{})
