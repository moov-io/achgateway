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

	// replace file
	err = chest.WriteFile("test-20210101-0101/foo.ach", []byte("nacha"))
	require.NoError(t, err)
	err = chest.ReplaceFile("test-20210101-0101/foo.ach", "after/foo.ach")
	require.NoError(t, err)
	_, err = chest.Open("after/foo.ach.canceled")
	require.NoError(t, err)

	// replace dir
	err = chest.ReplaceDir("after/", "final/")
	_, err = chest.Open("final/foo.ach.canceled")
	require.NoError(t, err)
}
