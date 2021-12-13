package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilesystem(t *testing.T) {
	dir := t.TempDir()
	chest, err := NewFilesystem(dir)
	require.NoError(t, err)

	testStorage(t, chest)
}
