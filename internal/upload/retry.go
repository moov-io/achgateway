// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/sethvargo/go-retry"
)

type RetryAgent struct {
	cfg        service.UploadRetry
	underlying Agent
}

func newRetryAgent(logger log.Logger, underlying Agent, cfg *service.UploadRetry) (*RetryAgent, error) {
	if cfg == nil {
		return nil, errors.New("nil UploadRetry config")
	}
	return &RetryAgent{
		cfg:        *cfg,
		underlying: underlying,
	}, nil
}

func (rt *RetryAgent) ID() string {
	return rt.underlying.ID()
}

func (rt *RetryAgent) String() string {
	return fmt.Sprintf("RetryAgent{%T}", rt.underlying)
}

func isRetryableError(err error) error {
	if err == nil {
		return nil
	}
	if os.IsTimeout(err) || strings.Contains(err.Error(), "no such host") {
		return retry.RetryableError(err)
	}
	return nil
}

func (rt *RetryAgent) newBackoff() (retry.Backoff, error) {
	fib := retry.NewFibonacci(rt.cfg.Interval)
	if fib == nil {
		return nil, fmt.Errorf("retry: problem creating %v as fibonacci", rt.cfg.Interval)
	}
	fib = retry.WithMaxRetries(rt.cfg.MaxRetries, fib)
	return fib, nil
}

func (rt *RetryAgent) retryFiles(f func() ([]string, error)) ([]string, error) {
	backoff, err := rt.newBackoff()
	if err != nil {
		return nil, err
	}
	var files []string
	ctx := context.Background()
	err = retry.Do(ctx, backoff, func(ctx context.Context) error {
		fs, err := f()
		if err := isRetryableError(err); err != nil {
			return err
		}
		files = fs
		return nil
	})
	return files, err
}

// Network'd calls
func (rt *RetryAgent) GetInboundFiles() ([]string, error) {
	return rt.retryFiles(rt.underlying.GetInboundFiles)
}

func (rt *RetryAgent) GetReconciliationFiles() ([]string, error) {
	return rt.retryFiles(rt.underlying.GetReconciliationFiles)
}

func (rt *RetryAgent) GetReturnFiles() ([]string, error) {
	return rt.retryFiles(rt.underlying.GetReturnFiles)
}

func (rt *RetryAgent) UploadFile(f File) error {
	backoff, err := rt.newBackoff()
	if err != nil {
		return err
	}
	ctx := context.Background()
	return retry.Do(ctx, backoff, func(ctx context.Context) error {
		return isRetryableError(rt.underlying.UploadFile(f))
	})
}

func (rt *RetryAgent) Delete(path string) error {
	backoff, err := rt.newBackoff()
	if err != nil {
		return err
	}
	ctx := context.Background()
	return retry.Do(ctx, backoff, func(ctx context.Context) error {
		return isRetryableError(rt.underlying.Delete(path))
	})
}

func (rt *RetryAgent) ReadFile(path string) (*File, error) {
	backoff, err := rt.newBackoff()
	if err != nil {
		return nil, err
	}
	var file *File
	ctx := context.Background()
	err = retry.Do(ctx, backoff, func(ctx context.Context) error {
		file, err = rt.underlying.ReadFile(path)
		if err := isRetryableError(err); err != nil {
			return err
		}
		return nil
	})
	return file, err
}

// Non-Network calls, so pass-through
func (rt *RetryAgent) InboundPath() string {
	return rt.underlying.InboundPath()
}

func (rt *RetryAgent) OutboundPath() string {
	return rt.underlying.OutboundPath()
}

func (rt *RetryAgent) ReconciliationPath() string {
	return rt.underlying.ReconciliationPath()
}

func (rt *RetryAgent) ReturnPath() string {
	return rt.underlying.ReturnPath()
}

func (rt *RetryAgent) Hostname() string {
	return rt.underlying.Hostname()
}

// Network calls, but direct pass-through

func (rt *RetryAgent) Ping() error {
	return rt.underlying.Ping()
}

func (rt *RetryAgent) Close() error {
	return rt.underlying.Close()
}
