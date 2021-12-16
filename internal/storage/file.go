package storage

import (
	"os"
	"time"
)

type File interface {
	Filename() string
	FullPath() string

	Read([]byte) (int, error)
	Close() error
}

type file struct {
	*os.File

	filename string
	fullpath string
}

func (f *file) Filename() string {
	return f.filename
}

func (f *file) FullPath() string {
	return f.fullpath
}

var _ File = (&file{})

type FileStat struct {
	RelativePath string
	ModTime      time.Time
}
