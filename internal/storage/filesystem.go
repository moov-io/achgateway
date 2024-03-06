package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type filesystem struct {
	root string
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
		root: fullpath,
	}, nil
}

func (fs *filesystem) Open(path string) (File, error) {
	if strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
		return nil, errors.New("invalid path")
	}

	path = filepath.Join(fs.root, path)

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

func (fs *filesystem) Glob(pattern string) ([]FileStat, error) {
	matches, err := filepath.Glob(filepath.Join(fs.root, pattern))
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
			RelativePath: strings.TrimPrefix(matches[i], fs.root+"/"),
			ModTime:      stat.ModTime(),
		})
	}
	return out, nil
}

func (fs *filesystem) ReplaceFile(oldpath, newpath string) error {
	oldpath = filepath.Join(fs.root, oldpath)
	newpath = filepath.Join(fs.root, newpath)

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

func (fs *filesystem) ReplaceDir(oldpath, newpath string) error {
	path := filepath.Join(fs.root, oldpath)

	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		// dir doesn't exist, so write newpath
		return os.MkdirAll(filepath.Join(fs.root, newpath), 0777)
	}

	// move the existing file
	return os.Rename(path, filepath.Join(fs.root, newpath))
}

func (fs *filesystem) MkdirAll(path string) error {
	return os.MkdirAll(filepath.Join(fs.root, path), 0777)
}

func (fs *filesystem) RmdirAll(path string) error {
	return os.RemoveAll(filepath.Join(fs.root, path))
}

func (fs *filesystem) WriteFile(path string, contents []byte) error {
	dir, path := filepath.Split(path)
	dir = filepath.Join(fs.root, dir)

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
