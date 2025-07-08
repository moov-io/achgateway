// Licensed to The Moov Authors under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. The Moov Authors licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pipeline

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/base/log"
	"github.com/stretchr/testify/require"
)

func TestCleanupService_NewCleanupService(t *testing.T) {
	logger := log.NewTestLogger()
	storage, err := storage.NewFilesystem(t.TempDir())
	require.NoError(t, err)

	shard := service.Shard{
		Name: "test-shard",
	}

	// Test with nil config
	cs, err := NewCleanupService(logger, storage, shard, nil)
	require.NoError(t, err)
	require.Nil(t, cs)

	// Test with disabled config
	config := &service.CleanupConfig{
		Enabled: false,
	}
	cs, err = NewCleanupService(logger, storage, shard, config)
	require.NoError(t, err)
	require.Nil(t, cs)

	// Test with enabled config
	config = &service.CleanupConfig{
		Enabled:           true,
		RetentionDuration: 24 * time.Hour,
		CheckInterval:     1 * time.Hour,
	}
	cs, err = NewCleanupService(logger, storage, shard, config)
	require.NoError(t, err)
	require.NotNil(t, cs)
	require.NotNil(t, cs.directoryPattern)
}

func TestCleanupService_shouldDeleteDirectory(t *testing.T) {
	logger := log.NewTestLogger()
	tempDir := t.TempDir()
	storage, err := storage.NewFilesystem(tempDir)
	require.NoError(t, err)

	shard := service.Shard{
		Name: "test-shard",
	}
	config := &service.CleanupConfig{
		Enabled:           true,
		RetentionDuration: 24 * time.Hour,
		CheckInterval:     1 * time.Hour,
	}

	cs, err := NewCleanupService(logger, storage, shard, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create test directories
	now := time.Now()
	oldDirName := fmt.Sprintf("%s-%s", shard.Name, now.Add(-48*time.Hour).Format("20060102-150405"))
	newDirName := fmt.Sprintf("%s-%s", shard.Name, now.Add(-1*time.Hour).Format("20060102-150405"))

	// Create old directory with uploaded files
	oldUploadedPath := path.Join(oldDirName, "uploaded")
	err = storage.MkdirAll(oldUploadedPath)
	require.NoError(t, err)
	err = storage.WriteFile(path.Join(oldUploadedPath, "test.ach"), []byte("test content"))
	require.NoError(t, err)

	// Create new directory with uploaded files
	newUploadedPath := path.Join(newDirName, "uploaded")
	err = storage.MkdirAll(newUploadedPath)
	require.NoError(t, err)
	err = storage.WriteFile(path.Join(newUploadedPath, "test.ach"), []byte("test content"))
	require.NoError(t, err)

	// Create directory without uploaded subdirectory (use different timestamp to avoid conflict)
	nouploadDirName := fmt.Sprintf("%s-%s", shard.Name, now.Add(-36*time.Hour).Format("20060102-150405"))
	err = storage.MkdirAll(nouploadDirName)
	require.NoError(t, err)

	cutoffTime := now.Add(-config.RetentionDuration)

	// Test old directory with uploaded files - should delete
	shouldDelete, err := cs.shouldDeleteDirectory(ctx, oldDirName, cutoffTime)
	require.NoError(t, err)
	require.True(t, shouldDelete)

	// Test new directory - should not delete (too new)
	shouldDelete, err = cs.shouldDeleteDirectory(ctx, newDirName, cutoffTime)
	require.NoError(t, err)
	require.False(t, shouldDelete)

	// Test directory without uploaded subdirectory - should not delete
	shouldDelete, err = cs.shouldDeleteDirectory(ctx, nouploadDirName, cutoffTime)
	require.NoError(t, err)
	require.False(t, shouldDelete)

	// Test invalid directory name format
	_, err = cs.shouldDeleteDirectory(ctx, "invalid-dir-name", cutoffTime)
	require.Error(t, err)
}

func TestCleanupService_runCleanup(t *testing.T) {
	logger := log.NewTestLogger()
	tempDir := t.TempDir()
	storage, err := storage.NewFilesystem(tempDir)
	require.NoError(t, err)

	shard := service.Shard{
		Name: "test-shard",
	}
	config := &service.CleanupConfig{
		Enabled:           true,
		RetentionDuration: 24 * time.Hour,
		CheckInterval:     1 * time.Hour,
	}

	cs, err := NewCleanupService(logger, storage, shard, config)
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Create directories to test
	// 1. Old directory with uploaded files (should be deleted)
	oldDir1 := fmt.Sprintf("%s-%s", shard.Name, now.Add(-48*time.Hour).Format("20060102-150405"))
	oldUploadedPath1 := path.Join(oldDir1, "uploaded")
	err = storage.MkdirAll(oldUploadedPath1)
	require.NoError(t, err)
	err = storage.WriteFile(path.Join(oldUploadedPath1, "test1.ach"), []byte("test content 1"))
	require.NoError(t, err)

	// 2. Another old directory with uploaded files (should be deleted)
	oldDir2 := fmt.Sprintf("%s-%s", shard.Name, now.Add(-72*time.Hour).Format("20060102-150405"))
	oldUploadedPath2 := path.Join(oldDir2, "uploaded")
	err = storage.MkdirAll(oldUploadedPath2)
	require.NoError(t, err)
	err = storage.WriteFile(path.Join(oldUploadedPath2, "test2.ach"), []byte("test content 2"))
	require.NoError(t, err)

	// 3. New directory (should not be deleted)
	newDir := fmt.Sprintf("%s-%s", shard.Name, now.Add(-1*time.Hour).Format("20060102-150405"))
	newUploadedPath := path.Join(newDir, "uploaded")
	err = storage.MkdirAll(newUploadedPath)
	require.NoError(t, err)
	err = storage.WriteFile(path.Join(newUploadedPath, "test3.ach"), []byte("test content 3"))
	require.NoError(t, err)

	// 4. Old directory without uploaded files (should not be deleted)
	oldDirNoUpload := fmt.Sprintf("%s-%s", shard.Name, now.Add(-36*time.Hour).Format("20060102-150405"))
	err = storage.MkdirAll(oldDirNoUpload)
	require.NoError(t, err)

	// 5. Directory that doesn't match pattern (should be ignored)
	ignoredDir := "some-other-directory"
	err = storage.MkdirAll(ignoredDir)
	require.NoError(t, err)

	// 6. Regular file (should be ignored)
	err = storage.WriteFile("regular-file.txt", []byte("test"))
	require.NoError(t, err)

	// Run cleanup
	cs.runCleanup(ctx)

	// Verify results
	entries, err := storage.ReadDir(".")
	require.NoError(t, err)

	dirNames := make(map[string]bool)
	for _, entry := range entries {
		dirNames[entry.Name()] = true
	}

	// Old directories with uploaded files should be deleted
	require.False(t, dirNames[oldDir1], "oldDir1 should have been deleted")
	require.False(t, dirNames[oldDir2], "oldDir2 should have been deleted")

	// These should still exist
	require.True(t, dirNames[newDir], "newDir should still exist")
	require.True(t, dirNames[oldDirNoUpload], "oldDirNoUpload should still exist")
	require.True(t, dirNames[ignoredDir], "ignoredDir should still exist")
	require.True(t, dirNames["regular-file.txt"], "regular-file.txt should still exist")
}

func TestCleanupService_GetStats(t *testing.T) {
	logger := log.NewTestLogger()
	tempDir := t.TempDir()
	storage, err := storage.NewFilesystem(tempDir)
	require.NoError(t, err)

	shard := service.Shard{
		Name: "test-shard",
	}
	config := &service.CleanupConfig{
		Enabled:           true,
		RetentionDuration: 24 * time.Hour,
		CheckInterval:     1 * time.Hour,
	}

	cs, err := NewCleanupService(logger, storage, shard, config)
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Create test directories
	// Old directory eligible for deletion
	oldDir := fmt.Sprintf("%s-%s", shard.Name, now.Add(-48*time.Hour).Format("20060102-150405"))
	oldUploadedPath := path.Join(oldDir, "uploaded")
	err = storage.MkdirAll(oldUploadedPath)
	require.NoError(t, err)
	err = storage.WriteFile(path.Join(oldUploadedPath, "test.ach"), []byte("test content"))
	require.NoError(t, err)

	// New directory not eligible
	newDir := fmt.Sprintf("%s-%s", shard.Name, now.Add(-1*time.Hour).Format("20060102-150405"))
	err = storage.MkdirAll(newDir)
	require.NoError(t, err)

	// Get stats
	stats, err := cs.GetStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	require.Equal(t, shard.Name, stats.ShardName)
	require.Equal(t, 2, stats.TotalDirectories)
	require.Equal(t, 1, stats.EligibleForDeletion)
	require.Equal(t, config.RetentionDuration, stats.RetentionDuration)
}

func TestCleanupService_StartStop(t *testing.T) {
	logger := log.NewTestLogger()
	tempDir := t.TempDir()
	storage, err := storage.NewFilesystem(tempDir)
	require.NoError(t, err)

	shard := service.Shard{
		Name: "test-shard",
	}
	config := &service.CleanupConfig{
		Enabled:           true,
		RetentionDuration: 24 * time.Hour,
		CheckInterval:     100 * time.Millisecond, // Short interval for testing
	}

	cs, err := NewCleanupService(logger, storage, shard, config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the service
	cs.Start(ctx)

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Stop the service
	cs.Stop()

	// Verify it stopped
	select {
	case <-cs.done:
		// Good, it's closed
	case <-time.After(1 * time.Second):
		t.Fatal("cleanup service did not stop in time")
	}
}
