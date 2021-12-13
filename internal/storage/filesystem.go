package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
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
	fd, err := os.Open(filepath.Join(fs.root, path))
	if err != nil {
		return nil, err
	}

	_, name := filepath.Split(path)

	return &file{
		filename: name,
		fullpath: fd.Name(),
	}, nil
}

func (fs *filesystem) Glob(pattern string) ([]string, error) {
	return filepath.Glob(filepath.Join(fs.root, pattern))
}

func (fs *filesystem) ReplaceFile(oldpath, newpath string) error {
	path := filepath.Join(fs.root, oldpath)

	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		// file doesn't exist, so write newpath
		return ioutil.WriteFile(filepath.Join(fs.root, newpath), nil, 0600)
	}

	// move the existing file
	return os.Rename(path, path+".canceled")
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

	return ioutil.WriteFile(filepath.Join(dir, path), contents, 0600)
}
