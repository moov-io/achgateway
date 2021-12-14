package storage

type Config struct {
	Filesystem FilesystemConfig
	Encryption EncryptionConfig
}

type FilesystemConfig struct {
	Directory string
}

type EncryptionConfig struct {
	AES      *AESConfig
	Encoding string
}

type AESConfig struct {
	Base64Key string
}
