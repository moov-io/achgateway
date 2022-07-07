package storage

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func testStorage(t *testing.T, chest Chest) {
	t.Helper()

	// read / write testing
	file, err := chest.Open("foo/bar.txt")
	require.Error(t, err)
	require.Nil(t, file)

	err = chest.WriteFile("foo/bar.txt", []byte("hello, world"))
	require.NoError(t, err)

	file, err = chest.Open("foo/bar.txt")
	require.NoError(t, err)

	bs, err := ioutil.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, bs, []byte("hello, world"))
	require.NoError(t, file.Close())

	// replace file
	err = chest.WriteFile("test-20210101-0101/foo.ach", []byte("nacha"))
	require.NoError(t, err)
	err = chest.ReplaceFile("test-20210101-0101/foo.ach", "after/foo.ach.canceled")
	require.NoError(t, err)
	file, err = chest.Open("after/foo.ach.canceled")
	require.NoError(t, err)
	require.NoError(t, file.Close())

	// replace dir
	err = chest.ReplaceDir("after/", "final/")
	require.NoError(t, err)
	file, err = chest.Open("final/foo.ach.canceled")
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

func readFinalContents(t *testing.T, chest Chest) string {
	t.Helper()

	file, err := chest.Open("final/foo.ach.canceled")
	require.NoError(t, err)

	defer func() {
		if file != nil {
			file.Close()
		}
	}()

	bs, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	return string(bs)
}
