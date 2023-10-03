package storage

import (
	"errors"
	"fmt"
	"io"
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
	var out []FileStat
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
		return os.WriteFile(newpath, nil, 0600)
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

	f, err := os.Create(filepath.Join(dir, path))
	if err != nil {
		return fmt.Errorf("WriteFile: %v", err)
	}
	defer f.Close()

	_, err = f.Write(contents)
	if err != nil {
		return fmt.Errorf("WriteFile: %v", err)
	}

	err = f.Sync()
	if err != nil {
		return fmt.Errorf("WriteFile: %v", err)
	}

	return nil
}

func (fs *filesystem) GetFileWriter(path string) (io.Writer, func() error, error) {
	dir, path := filepath.Split(path)
	dir = filepath.Join(fs.root, dir)

	if err := os.MkdirAll(dir, 0777); err != nil {
		return nil, nil, err
	}

	f, err := os.Create(filepath.Join(dir, path))
	if err != nil {
		return nil, nil, fmt.Errorf("GetFileWriter: %v", err)
	}

	continuation := func() error {
		defer f.Close()
		err = f.Sync()
		if err != nil {
			return fmt.Errorf("WriteFile: %v", err)
		}
		return nil
	}

	return f, continuation, nil
}
