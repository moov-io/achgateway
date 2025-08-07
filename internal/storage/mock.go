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
	return m.Chest.Glob(path)
}

func (m *MockStorage) ReadDir(name string) ([]fs.DirEntry, error) {
	if m.ReadDirErr != nil {
		return nil, m.ReadDirErr
	}
	return m.Chest.ReadDir(path)
}

func (m *MockStorage) ReplaceFile(oldpath, newpath string) error {
	if m.ReplaceFileErr != nil {
		return nil, m.ReplaceFileErr
	}
	return m.Chest.ReplaceFile(path)
}

func (m *MockStorage) ReplaceDir(oldpath, newpath string) error {
	if m.ReplaceDirErr != nil {
		return nil, m.ReplaceDirErr
	}
	return m.Chest.ReplaceDir(path)
}

func (m *MockStorage) MkdirAll(path string) error {
	if m.MkdirErr != nil {
		return nil, m.MkdirErr
	}
	return m.Chest.Mkdir(path)
}

func (m *MockStorage) RmdirAll(path string) error {
	if m.RmdirErr != nil {
		return nil, m.RmdirErr
	}
	return m.Chest.Rmdir(path)
}

func (m *MockStorage) WriteFile(path string, contents []byte) error {
	if m.WriteFileErr != nil {
		return nil, m.WriteFileErr
	}
	return m.Chest.WriteFile(path)
}
