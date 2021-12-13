package storage

import (
	"bytes"
)

type buffer struct {
	b *bytes.Buffer

	filename string
	fullpath string
}

func (b *buffer) Filename() string {
	return b.filename
}

func (b *buffer) FullPath() string {
	return b.fullpath
}

func (b *buffer) Read(data []byte) (int, error) {
	return b.b.Read(data)
}

func (b *buffer) Close() error {
	b.b.Reset()
	return nil
}

var _ File = (&buffer{})
