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
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/storage"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/telemetry"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	cleanupRunsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "achgateway_cleanup_runs_total",
		Help: "Total number of cleanup runs executed",
	}, []string{"shard", "status"})

	cleanupDirectoriesDeletedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "achgateway_cleanup_directories_deleted_total",
		Help: "Total number of directories deleted by cleanup",
	}, []string{"shard"})

	cleanupErrorsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "achgateway_cleanup_errors_total",
		Help: "Total number of errors during cleanup",
	}, []string{"shard", "error_type"})
)

// CleanupService manages the periodic cleanup of old processed ACH files
type CleanupService struct {
	logger  log.Logger
	storage storage.Chest
	shard   service.Shard
	config  *service.CleanupConfig
	ticker  *time.Ticker
	done    chan struct{}

	// directoryPattern matches directories created by isolateMergableDir
	// Format: <shard-name>-YYYYMMDD-HHMMSS
	directoryPattern *regexp.Regexp
}

// NewCleanupService creates a new cleanup service for a shard
func NewCleanupService(logger log.Logger, storage storage.Chest, shard service.Shard, config *service.CleanupConfig) (*CleanupService, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}

	// Create pattern to match isolated directories
	pattern := fmt.Sprintf("^%s-\\d{8}-\\d{6}$", regexp.QuoteMeta(shard.Name))
	dirPattern, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile directory pattern: %w", err)
	}

	return &CleanupService{
		logger:           logger,
		storage:          storage,
		shard:            shard,
		config:           config,
		directoryPattern: dirPattern,
		done:             make(chan struct{}),
	}, nil
}

// Start begins the periodic cleanup process
func (cs *CleanupService) Start(ctx context.Context) {
	if cs == nil {
		return
	}

	cs.ticker = time.NewTicker(cs.config.CheckInterval)

	cs.logger.Info().With(log.Fields{
		"shard":             log.String(cs.shard.Name),
		"checkInterval":     log.String(cs.config.CheckInterval.String()),
		"retentionDuration": log.String(cs.config.RetentionDuration.String()),
	}).Log("starting cleanup service")

	// Run initial cleanup
	cs.runCleanup(ctx)

	go func() {
		for {
			select {
			case <-cs.ticker.C:
				cs.runCleanup(ctx)
			case <-ctx.Done():
				cs.Stop()
				return
			case <-cs.done:
				return
			}
		}
	}()
}

// Stop halts the cleanup service
func (cs *CleanupService) Stop() {
	if cs == nil {
		return
	}

	cs.logger.Info().With(log.Fields{
		"shard": log.String(cs.shard.Name),
	}).Log("stopping cleanup service")

	if cs.ticker != nil {
		cs.ticker.Stop()
	}
	close(cs.done)
}

// runCleanup performs a single cleanup run
func (cs *CleanupService) runCleanup(ctx context.Context) {
	ctx, span := telemetry.StartSpan(ctx, "cleanup-run", trace.WithAttributes(
		attribute.String("achgateway.shard", cs.shard.Name),
	))
	defer span.End()

	cs.logger.Debug().With(log.Fields{
		"shard": log.String(cs.shard.Name),
	}).Log("starting cleanup run")

	startTime := time.Now()
	deletedCount := 0
	errorCount := 0

	// List all directories in the storage root
	entries, err := cs.storage.ReadDir(".")
	if err != nil {
		cs.logger.Error().With(log.Fields{
			"shard": log.String(cs.shard.Name),
		}).LogErrorf("failed to read storage directory: %v", err)
		cleanupRunsCounter.WithLabelValues(cs.shard.Name, "error").Inc()
		cleanupErrorsCounter.WithLabelValues(cs.shard.Name, "read_dir").Inc()
		span.RecordError(err)
		return
	}

	cutoffTime := time.Now().Add(-cs.config.RetentionDuration)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()

		// Skip if directory doesn't match our pattern
		if !cs.directoryPattern.MatchString(dirName) {
			continue
		}

		logger := cs.logger.Warn().With(log.Fields{
			"shard":     log.String(cs.shard.Name),
			"directory": log.String(dirName),
		})

		// Check if directory should be deleted
		shouldDelete, err := cs.shouldDeleteDirectory(ctx, dirName, cutoffTime)
		switch {
		case err != nil:
			logger.Error().Logf("error checking directory: %v", err)

			errorCount++
			cleanupErrorsCounter.WithLabelValues(cs.shard.Name, "check_directory").Inc()

		case shouldDelete:
			if err := cs.deleteDirectory(ctx, dirName); err != nil {
				logger.Error().LogErrorf("failed to delete directory: %v", err)

				errorCount++
				cleanupErrorsCounter.WithLabelValues(cs.shard.Name, "delete_directory").Inc()
			} else {
				deletedCount++
				cleanupDirectoriesDeletedCounter.WithLabelValues(cs.shard.Name).Inc()
			}
		}
	}

	duration := time.Since(startTime)

	logger := cs.logger.With(log.Fields{
		"shard":    log.String(cs.shard.Name),
		"deleted":  log.Int(deletedCount),
		"errors":   log.Int(errorCount),
		"duration": log.String(duration.String()),
	})
	logger.Info().Log("completed cleanup run")

	span.SetAttributes(
		attribute.Int("achgateway.cleanup.deleted_count", deletedCount),
		attribute.Int("achgateway.cleanup.error_count", errorCount),
		attribute.String("achgateway.cleanup.duration", duration.String()),
	)

	if errorCount == 0 {
		cleanupRunsCounter.WithLabelValues(cs.shard.Name, "success").Inc()
	} else {
		cleanupRunsCounter.WithLabelValues(cs.shard.Name, "partial_error").Inc()
	}
}

// shouldDeleteDirectory determines if a directory should be deleted based on:
// 1. Directory age (older than retention duration)
// 2. Presence of uploaded/ subdirectory (indicates successful processing)
// 3. Directory timestamp parsed from name
func (cs *CleanupService) shouldDeleteDirectory(ctx context.Context, dirName string, cutoffTime time.Time) (bool, error) {
	_, span := telemetry.StartSpan(ctx, "check-directory", trace.WithAttributes(
		attribute.String("achgateway.shard", cs.shard.Name),
		attribute.String("achgateway.directory", dirName),
	))
	defer span.End()

	// Extract just the directory name in case we got a full path
	// This is important for Windows compatibility where dirName might be a full path
	baseName := filepath.Base(dirName)

	// Parse timestamp from directory name
	// Format: <shard-name>-YYYYMMDD-HHMMSS
	if len(baseName) <= len(cs.shard.Name)+1 {
		return false, fmt.Errorf("directory name too short: %s", baseName)
	}

	timestampStr := baseName[len(cs.shard.Name)+1:] // Skip shard name and hyphen
	dirTime, err := time.Parse("20060102-150405", timestampStr)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to parse directory timestamp from %s: %w", baseName, err)
	}

	// Check if directory is old enough
	if dirTime.After(cutoffTime) {
		span.SetAttributes(
			attribute.Bool("achgateway.cleanup.too_new", true),
			attribute.String("achgateway.cleanup.dir_time", dirTime.String()),
			attribute.String("achgateway.cleanup.cutoff_time", cutoffTime.String()),
		)
		return false, nil
	}

	// Check for uploaded/ subdirectory
	uploadedPath := filepath.Join(dirName, "uploaded")
	entries, err := cs.storage.ReadDir(uploadedPath)
	if err != nil {
		// If we can't read the uploaded directory, it might not exist
		// which means the files weren't successfully processed
		span.SetAttributes(attribute.Bool("achgateway.cleanup.uploaded_dir_exists", false))
		return false, nil
	}

	// Ensure there are files in the uploaded directory
	hasUploadedFiles := false
	for _, entry := range entries {
		if !entry.IsDir() {
			hasUploadedFiles = true
			break
		}
	}

	span.SetAttributes(
		attribute.Bool("achgateway.cleanup.uploaded_dir_exists", true),
		attribute.Bool("achgateway.cleanup.has_uploaded_files", hasUploadedFiles),
		attribute.String("achgateway.cleanup.dir_time", dirTime.String()),
		attribute.String("achgateway.cleanup.cutoff_time", cutoffTime.String()),
	)

	return hasUploadedFiles, nil
}

// deleteDirectory removes a directory and all its contents
func (cs *CleanupService) deleteDirectory(ctx context.Context, dirName string) error {
	_, span := telemetry.StartSpan(ctx, "delete-directory", trace.WithAttributes(
		attribute.String("achgateway.shard", cs.shard.Name),
		attribute.String("achgateway.directory", dirName),
	))
	defer span.End()

	cs.logger.Info().With(log.Fields{
		"shard":     log.String(cs.shard.Name),
		"directory": log.String(dirName),
	}).Log("deleting directory")

	err := cs.storage.RmdirAll(dirName)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}

// GetStats returns current statistics about directories that could be cleaned up
func (cs *CleanupService) GetStats(ctx context.Context) (*CleanupStats, error) {
	if cs == nil {
		return nil, nil
	}

	stats := &CleanupStats{
		ShardName:         cs.shard.Name,
		RetentionDuration: cs.config.RetentionDuration,
	}

	entries, err := cs.storage.ReadDir(".")
	if err != nil {
		wd, _ := os.Getwd()

		return nil, fmt.Errorf("reading %s failed: %w", wd, err)
	}

	cutoffTime := time.Now().Add(-cs.config.RetentionDuration)

	for _, entry := range entries {
		if !entry.IsDir() || !cs.directoryPattern.MatchString(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("getting info on %s failed: %w", entry.Name(), err)
		}

		stats.TotalDirectories++

		shouldDelete, err := cs.shouldDeleteDirectory(ctx, entry.Name(), cutoffTime)
		if err == nil && shouldDelete {
			stats.EligibleForDeletion++
			stats.TotalSize += info.Size()
		}
		if err != nil {
			return nil, fmt.Errorf("checking %s to delete failed: %w", entry.Name(), err)
		}
	}

	return stats, nil
}

// CleanupStats contains statistics about cleanup operations
type CleanupStats struct {
	ShardName           string
	TotalDirectories    int
	EligibleForDeletion int
	TotalSize           int64
	RetentionDuration   time.Duration
}
