package storage

import (
	"os"
)

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
