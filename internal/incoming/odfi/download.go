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

package odfi

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/strx"
	"github.com/moov-io/base/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	filesDownloaded = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "files_downloaded",
		Help: "Counter of files downloaded from a remote server",
	}, []string{"kind"})
)

type Downloader interface {
	CopyFilesFromRemote(ctx context.Context, agent upload.Agent, shard *service.Shard) (*downloadedFiles, error)
}

func NewDownloader(logger log.Logger, cfg service.ODFIStorage) (Downloader, error) {
	baseDir := strx.Or(cfg.Directory, "storage")
	if err := os.MkdirAll(baseDir, 0777); err != nil {
		return nil, fmt.Errorf("problem creating %s: %v", baseDir, err)
	}
	return &downloaderImpl{
		logger:  logger,
		baseDir: baseDir,
	}, nil
}

type downloaderImpl struct {
	logger  log.Logger
	baseDir string
}

// downloadedFiles is a randomly generated directory inside of the storage directory.
// These are designed to be deleted after all files are processed.
type downloadedFiles struct {
	dir string
}

func (d *downloadedFiles) deleteFiles() error {
	return os.RemoveAll(d.dir)
}

func (d *downloadedFiles) deleteEmptyDirs(ctx context.Context, agent upload.Agent) error {
	count := func(ctx context.Context, path string) int {
		infos, err := os.ReadDir(path)
		if err != nil {
			return -1
		}

		_, span := telemetry.StartSpan(ctx, "odfi-delete-empty-dirs", trace.WithAttributes(
			attribute.String("achgateway.path", path),
			attribute.Int("achgateway.files", len(infos)),
		))
		defer span.End()

		return len(infos)
	}
	if path := filepath.Join(d.dir, agent.InboundPath()); count(ctx, path) == 0 {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("delete inbound: %v", err)
		}
	}
	if path := filepath.Join(d.dir, agent.ReconciliationPath()); count(ctx, path) == 0 {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("delete reconciliation: %v", err)
		}
	}
	if path := filepath.Join(d.dir, agent.ReturnPath()); count(ctx, path) == 0 {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("delete return: %v", err)
		}
	}
	return nil
}

func (dl *downloaderImpl) setup(agent upload.Agent) (*downloadedFiles, error) {
	dir, err := os.MkdirTemp(dl.baseDir, "download")
	if err != nil {
		return nil, err
	}

	dl.logger.Logf("created directory %s", dir)

	// Create sub-directories for files we download
	path := filepath.Join(dir, agent.InboundPath())
	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, fmt.Errorf("problem creating %s: %v", path, err)
	}
	path = filepath.Join(dir, agent.ReconciliationPath())
	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, fmt.Errorf("problem creating %s: %v", path, err)
	}
	path = filepath.Join(dir, agent.ReturnPath())
	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, fmt.Errorf("problem creating %s: %v", path, err)
	}

	return &downloadedFiles{
		dir: dir,
	}, nil
}

func (dl *downloaderImpl) CopyFilesFromRemote(ctx context.Context, agent upload.Agent, shard *service.Shard) (*downloadedFiles, error) {
	out, err := dl.setup(agent)
	if err != nil {
		return nil, err
	}

	logger := dl.logger.With(log.Fields{
		"shard": log.String(shard.Name),
	})

	// copy down files from our "inbound" directory
	filepaths, err := agent.GetInboundFiles(ctx)
	logger.Logf("%T found %d inbound files in %s", agent, len(filepaths), agent.InboundPath())
	if err != nil {
		return out, fmt.Errorf("problem downloading inbound files: %v", err)
	}
	filesDownloaded.With("kind", "inbound").Add(float64(len(filepaths)))
	if err := saveFilepaths(ctx, logger, agent, filepaths, filepath.Join(out.dir, agent.InboundPath())); err != nil {
		return out, fmt.Errorf("problem saving inbound files: %v", err)
	}

	// copy down files from out "reconciliation" directory
	filepaths, err = agent.GetReconciliationFiles(ctx)
	logger.Logf("%T found %d reconciliation files in %s", agent, len(filepaths), agent.ReconciliationPath())
	if err != nil {
		return out, fmt.Errorf("problem downloading reconciliation files: %v", err)
	}
	filesDownloaded.With("kind", "reconciliation").Add(float64(len(filepaths)))
	if err := saveFilepaths(ctx, logger, agent, filepaths, filepath.Join(out.dir, agent.ReconciliationPath())); err != nil {
		return out, fmt.Errorf("problem saving reconciliation files: %v", err)
	}

	// copy down files from out "return" directory
	filepaths, err = agent.GetReturnFiles(ctx)
	logger.Logf("%T found %d return files in %s", agent, len(filepaths), agent.ReturnPath())
	if err != nil {
		return out, fmt.Errorf("problem downloading return files: %v", err)
	}
	filesDownloaded.With("kind", "return").Add(float64(len(filepaths)))
	if err := saveFilepaths(ctx, logger, agent, filepaths, filepath.Join(out.dir, agent.ReturnPath())); err != nil {
		return out, fmt.Errorf("problem saving return files: %v", err)
	}

	return out, nil
}

// saveFilepaths will create files in dir for each file object provided
// The contents of each file struct will always be closed.
func saveFilepaths(ctx context.Context, logger log.Logger, agent upload.Agent, filepaths []string, dir string) error {
	var firstErr error
	var errordFilenames []string

	os.MkdirAll(dir, 0777) // ignore errors
	for i := range filepaths {
		outPath := filepath.Join(dir, filepath.Base(filepaths[i]))
		f, err := os.Create(outPath)
		if err != nil {
			err = fmt.Errorf("os.create on %s failed: %w", outPath, err)
			if firstErr == nil {
				firstErr = err
			} else {
				logger.Error().LogError(err)
			}
			errordFilenames = append(errordFilenames, filepaths[i])
			continue
		}

		file, err := agent.ReadFile(ctx, filepaths[i])
		if err != nil {
			// Save the error if it's our first, otherwise log
			err = fmt.Errorf("reading %s failed: %w", filepaths[i], err)
			if firstErr == nil {
				firstErr = err
			} else {
				logger.Error().LogError(err)
			}
		}
		if file == nil || err != nil {
			// Record the failure and skip copy
			errordFilenames = append(errordFilenames, filepaths[i])
			continue
		}
		if _, err = io.Copy(f, file.Contents); err != nil {
			err = fmt.Errorf("copying %s failed: %w", file.Filepath, err)
			if firstErr == nil {
				firstErr = err
			} else {
				logger.Error().LogError(err)
			}
			errordFilenames = append(errordFilenames, filepaths[i])
		}
		if err := f.Sync(); err != nil {
			return fmt.Errorf("sync on %s failed: %w", file.Filepath, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close on %s failed: %w", file.Filepath, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("closing %s had problem: %w", file.Filepath, err)
		}
		logger.Logf("saved %s at %s", filepaths[i], outPath)
	}
	if len(errordFilenames) != 0 {
		return fmt.Errorf("saveFilepaths problem on: %s: %v", strings.Join(errordFilenames, ", "), firstErr)
	}
	return nil
}
