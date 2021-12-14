package storage

import (
	"testing"

	"github.com/moov-io/cryptfs"

	"github.com/stretchr/testify/require"
)

func TestEncrypted(t *testing.T) {
	dir := t.TempDir()

	chest, err := NewFilesystem(dir)
	require.NoError(t, err)

	aes, err := cryptfs.NewAESCryptor([]byte("1234567887654321"))
	require.NoError(t, err)

	crypt, err := cryptfs.New(aes)
	require.NoError(t, err)
	crypt.SetCoder(cryptfs.Base64())

	testStorage(t, NewEncrypted(chest, crypt))

	finalContents := readFinalContents(t, chest)
	require.NotEqual(t, "nacha", finalContents)

	decrypted, err := crypt.Reveal([]byte(finalContents))
	require.NoError(t, err)
	require.Equal(t, []byte("nacha"), decrypted)
}
