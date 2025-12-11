package storage

import (
	"io/fs"
)

type MockStorage struct {
	Chest

	OpenErr error
	GlobErr error

	ReadDirErr error

	ReplaceFileErr error
	ReplaceDirErr  error

	MkdirAllErr error
	RmdirAllErr error

	WriteFileErr error
}

func (m *MockStorage) Open(path string) (fs.File, error) {
	if m.OpenErr != nil {
		return nil, m.OpenErr
	}
	return m.Chest.Open(path)
}

func (m *MockStorage) Glob(pattern string) ([]FileStat, error) {
	if m.GlobErr != nil {
		return nil, m.GlobErr
	}
	return m.Chest.Glob(pattern)
}

func (m *MockStorage) ReadDir(name string) ([]fs.DirEntry, error) {
	if m.ReadDirErr != nil {
		return nil, m.ReadDirErr
	}
	return m.Chest.ReadDir(name)
}

func (m *MockStorage) ReplaceFile(oldpath, newpath string) error {
	if m.ReplaceFileErr != nil {
		return m.ReplaceFileErr
	}
	return m.Chest.ReplaceFile(oldpath, newpath)
}

func (m *MockStorage) ReplaceDir(oldpath, newpath string) error {
	if m.ReplaceDirErr != nil {
		return m.ReplaceDirErr
	}
	return m.Chest.ReplaceDir(oldpath, newpath)
}

func (m *MockStorage) MkdirAll(path string) error {
	if m.MkdirAllErr != nil {
		return m.MkdirAllErr
	}
	return m.Chest.MkdirAll(path)
}

func (m *MockStorage) RmdirAll(path string) error {
	if m.RmdirAllErr != nil {
		return m.RmdirAllErr
	}
	return m.Chest.RmdirAll(path)
}

func (m *MockStorage) WriteFile(path string, contents []byte) error {
	if m.WriteFileErr != nil {
		return m.WriteFileErr
	}
	return m.Chest.WriteFile(path, contents)
}
