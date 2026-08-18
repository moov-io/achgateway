package storage

import (
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moov-io/base"
	"github.com/stretchr/testify/require"
)

func TestFilesystem(t *testing.T) {
	dir := t.TempDir()
	chest, err := NewFilesystem(dir)
	require.NoError(t, err)

	testStorage(t, chest)

	finalContents := readFinalContents(t, chest)
	require.Equal(t, "nacha", finalContents)
}

func TestFilesystemRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	chest, err := NewFilesystem(dir)
	require.NoError(t, err)

	parentEscape := filepath.Join(filepath.Dir(dir), "achgateway-escaped.ach")
	t.Cleanup(func() { os.Remove(parentEscape) })

	err = chest.WriteFile("../achgateway-escaped.ach", []byte("pwn"))
	require.EqualError(t, err, "invalid path")
	_, err = os.Stat(parentEscape)
	require.True(t, os.IsNotExist(err))

	err = chest.WriteFile("/tmp/achgateway-escaped.ach", []byte("pwn"))
	require.EqualError(t, err, "invalid path")

	err = chest.RmdirAll("..")
	require.EqualError(t, err, "invalid path")
	_, err = os.Stat(dir)
	require.NoError(t, err)

	err = chest.MkdirAll("../outside")
	require.EqualError(t, err, "invalid path")

	err = chest.ReplaceFile("../old", "ok")
	require.EqualError(t, err, "invalid path")
	err = chest.ReplaceFile("ok", "../new")
	require.EqualError(t, err, "invalid path")

	err = chest.ReplaceDir("../old", "ok")
	require.EqualError(t, err, "invalid path")

	_, err = chest.Glob("../../*")
	require.EqualError(t, err, "invalid path")

	_, err = chest.Open("../secret")
	require.EqualError(t, err, "invalid path")

	err = chest.WriteFile("mergable/SD1/f1.ach", []byte("nacha"))
	require.NoError(t, err)
	file, err := chest.Open("mergable/SD1/f1.ach")
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

func setupFilesystemGlobTest(tb testing.TB, iterations int) (Chest, string, int) {
	tb.Helper()

	start := time.Now()
	defer func() {
		tb.Logf("setupFilesystemGlobTest took: %v", time.Since(start))
	}()

	dir := tb.TempDir()
	sub := filepath.Join("a", "b")
	require.NoError(tb, os.MkdirAll(filepath.Join(dir, sub), 0777))

	chest, err := NewFilesystem(dir)
	require.NoError(tb, err)

	// Write files to directory
	var canceled atomic.Int32
	contents := []byte("ach file contents")

	var wg sync.WaitGroup
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer wg.Done()

			where := filepath.Join(sub, base.ID()+".ach")

			err := chest.WriteFile(where, contents)
			require.NoError(tb, err)

			// cancel the file one of two ways
			if i%10 == 0 {
				canceled.Add(1)

				if i%50 == 0 {
					// rename it
					err = chest.ReplaceFile(where, where+".canceled")
					require.NoError(tb, err)
				} else {
					// add .canceled file
					err = chest.WriteFile(where+".canceled", nil)
					require.NoError(tb, err)
				}
			}
		}(i)
	}
	wg.Wait()

	return chest, sub, int(canceled.Load())
}

func TestFilesystemGlob(t *testing.T) {
	chest, sub, canceled := setupFilesystemGlobTest(t, 514) // more than readdirChunkSize

	matches, err := chest.Glob(sub + "/*.canceled")
	require.NoError(t, err)
	require.Len(t, matches, canceled)
}

func BenchmarkFilesystem_Glob(b *testing.B) {
	b.Run("user homedir", func(b *testing.B) {
		b.Skip() // don't run dir sweeps over arbitrary homedirs

		who, err := user.Current()
		require.NoError(b, err)
		require.NotEmpty(b, who.HomeDir)

		chest, err := NewFilesystem(who.HomeDir)
		require.NoError(b, err)

		filename := "achgateway-filesystem-benchmark.ach.canceled"
		err = chest.WriteFile(filename, nil)
		require.NoError(b, err)
		b.Cleanup(func() {
			os.Remove(filepath.Join(who.HomeDir, "achgateway-filesystem-benchmark.ach.canceled"))
		})
		b.ResetTimer()

		matches, err := chest.Glob("/*.canceled")
		require.NoError(b, err)
		require.NotEmpty(b, matches)
	})

	b.Run("write files", func(b *testing.B) {
		b.Skip() // really slow to write files

		chest, sub, canceled := setupFilesystemGlobTest(b, b.N)
		b.ResetTimer()

		matches, err := chest.Glob(sub + "/*.canceled")
		require.NoError(b, err)
		require.Len(b, matches, canceled)
	})
}
