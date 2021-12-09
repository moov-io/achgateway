package storage

type Chest interface {
	Open(path string) (File, error)
	Glob(pattern string) ([]string, error)

	ReplaceFile(oldpath, newpath string) error
	ReplaceDir(oldpath, newpath string) error

	MkdirAll(path string) error
	RmdirAll(path string) error

	WriteFile(path string, contents []byte) error
}

type File interface {
	Filename() string
	FullPath() string

	Read([]byte) (int, error)
	Close() error
}
