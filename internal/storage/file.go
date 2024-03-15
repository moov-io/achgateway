package storage

import (
	"io/fs"
	"os"
	"time"
)

type File interface {
	Filename() string
	FullPath() string

	Stat() (fs.FileInfo, error)
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

var (
	_ fs.File = (&file{})
	_ File    = (&file{})
)

type FileStat struct {
	RelativePath string
	ModTime      time.Time
}
