package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type filesystem struct {
	root string
	fsys fs.FS
}

func NewFilesystem(root string) (Chest, error) {
	fullpath, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fullpath, 0777); err != nil {
		return nil, err
	}
	return &filesystem{
		fsys: os.DirFS(fullpath),
		root: fullpath,
	}, nil
}

func (f *filesystem) Open(path string) (fs.File, error) {
	if strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
		return nil, errors.New("invalid path")
	}

	path = filepath.Join(f.root, path)

	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	_, name := filepath.Split(path)

	return &file{
		File:     fd,
		filename: name,
		fullpath: fd.Name(),
	}, nil
}

func (f *filesystem) ReadDir(name string) ([]fs.DirEntry, error) {
	if rd, ok := f.fsys.(fs.ReadDirFS); ok {
		return rd.ReadDir(name)
	}
	return os.ReadDir(name)
}

var _ fs.ReadDirFS = (&filesystem{})

func (f *filesystem) Glob(pattern string) ([]FileStat, error) {
	matches, err := filepath.Glob(filepath.Join(f.root, pattern))
	if err != nil {
		return nil, err
	}
	out := make([]FileStat, 0, len(matches))
	for i := range matches {
		stat, _ := os.Stat(matches[i])
		if stat == nil {
			continue
		}
		out = append(out, FileStat{
			RelativePath: strings.TrimPrefix(matches[i], f.root+"/"),
			ModTime:      stat.ModTime(),
		})
	}
	return out, nil
}

func (f *filesystem) ReplaceFile(oldpath, newpath string) error {
	oldpath = filepath.Join(f.root, oldpath)
	newpath = filepath.Join(f.root, newpath)

	// Create the new dir(s)
	dir, _ := filepath.Split(newpath)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	// file doesn't exist, so write newpath
	if _, err := os.Stat(oldpath); err != nil && os.IsNotExist(err) {
		return write(newpath, nil)
	}

	// move the existing file
	return os.Rename(oldpath, newpath)
}

func (f *filesystem) ReplaceDir(oldpath, newpath string) error {
	path := filepath.Join(f.root, oldpath)

	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		// dir doesn't exist, so write newpath
		return os.MkdirAll(filepath.Join(f.root, newpath), 0777)
	}

	// move the existing file
	return os.Rename(path, filepath.Join(f.root, newpath))
}

func (f *filesystem) MkdirAll(path string) error {
	return os.MkdirAll(filepath.Join(f.root, path), 0777)
}

func (f *filesystem) RmdirAll(path string) error {
	return os.RemoveAll(filepath.Join(f.root, path))
}

func (f *filesystem) WriteFile(path string, contents []byte) error {
	dir, path := filepath.Split(path)
	dir = filepath.Join(f.root, dir)

	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	return write(filepath.Join(dir, path), contents)
}

func write(where string, data []byte) error {
	fd, err := os.OpenFile(where, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating %s failed: %w", where, err)
	}
	defer fd.Close()

	_, err = fd.Write(data)
	if err != nil {
		return fmt.Errorf("writing %s failed: %w", where, err)
	}

	return fd.Sync()
}
