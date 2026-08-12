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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestEmitFilesUploaded_BoundedConcurrency(t *testing.T) {
	// More events than fileUploadedEmitLimit must all still emit, while
	// concurrent Send calls stay within the limit.
	const n = fileUploadedEmitLimit*2 + 17

	// Gate: first wave of workers block until we have observed enough
	// concurrency, then release so the rest can finish. Deterministic —
	// no wall-clock sleep.
	var (
		mu          sync.Mutex
		inFlight    int
		maxInFlight int
		totalSent   int
		held        int // how many workers are parked on the barrier
		release     = make(chan struct{})
		// first wave signals when it has entered Send
		entered sync.WaitGroup
	)
	entered.Add(fileUploadedEmitLimit)

	emitter := &concurrencyTrackingEmitter{
		onSend: func() {
			mu.Lock()
			inFlight++
			totalSent++
			if inFlight > maxInFlight {
				maxInFlight = inFlight
			}
			// Park only the first wave so the limit is fully occupied.
			park := held < fileUploadedEmitLimit
			if park {
				held++
			}
			mu.Unlock()

			if park {
				entered.Done()
				<-release // hold the slot until the test opens the gate
			}

			mu.Lock()
			inFlight--
			mu.Unlock()
		},
	}

	inputs := make([]string, n)
	for i := range inputs {
		inputs[i] = fmt.Sprintf("file-%04d.ach", i)
	}
	xfagg := &aggregator{
		logger:       log.NewTestLogger(),
		eventEmitter: emitter,
		shard:        service.Shard{Name: "test-emit-limit"},
	}

	done := make(chan error, 1)
	go func() {
		done <- xfagg.emitFilesUploaded(context.Background(), mergedFiles{
			{
				InputFilepaths:   inputs,
				UploadedFilename: "merged.ach",
				Shard:            "test-emit-limit",
			},
		})
	}()

	// Wait on the test goroutine until the concurrency limit is saturated.
	waitDone := make(chan struct{})
	go func() {
		entered.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
		// proceed
	case err := <-done:
		require.FailNow(t, "emit finished before concurrency barrier filled", "err=%v", err)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for emit workers to reach concurrency limit")
	}

	mu.Lock()
	observed := maxInFlight
	mu.Unlock()
	require.Equal(t, fileUploadedEmitLimit, observed, "should saturate emit limit before release")

	close(release)
	require.NoError(t, <-done)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, n, totalSent, "every input must still get a FileUploaded")
	require.Equal(t, fileUploadedEmitLimit, maxInFlight)
}

// concurrencyTrackingEmitter is an events.Emitter that records concurrency.
type concurrencyTrackingEmitter struct {
	onSend func()
}

func (e *concurrencyTrackingEmitter) Send(context.Context, models.Event) error {
	if e.onSend != nil {
		e.onSend()
	}
	return nil
}

func TestAggregate_mergeAndEmitErrorsJoined(t *testing.T) {
	// When merge already failed, emit failures must still surface (errors.Join),
	// not be dropped because mergeErr was non-nil.
	partial := mergedFiles{
		{
			InputFilepaths:   []string{"a.ach"},
			UploadedFilename: "out.ach",
			Shard:            "test-join",
		},
	}
	mergeErr := errors.New("unmapped input files")
	emitErr := errors.New("kafka produce failed")

	emitter := &events.MockEmitter{}
	emitter.SetSendError(emitErr)

	xfagg := &aggregator{
		logger:       log.NewTestLogger(),
		eventEmitter: emitter,
		shard:        service.Shard{Name: "test-join"},
		merger: &MockXferMerging{
			merged: partial,
			Err:    mergeErr,
		},
	}

	t.Run("automated joins both errors", func(t *testing.T) {
		err := xfagg.withEachFile(time.Now())
		require.Error(t, err)
		require.ErrorContains(t, err, "unmapped input files")
		require.ErrorContains(t, err, "kafka produce failed")
		require.ErrorIs(t, err, mergeErr)
		require.ErrorIs(t, err, emitErr)
	})

	t.Run("manual joins both errors", func(t *testing.T) {
		waiter := manuallyTriggeredCutoff{
			ctx: context.Background(),
			C:   make(chan error, 1),
		}
		xfagg.manualCutoff(waiter)
		err := <-waiter.C
		require.Error(t, err)
		require.ErrorContains(t, err, "unmapped input files")
		require.ErrorContains(t, err, "kafka produce failed")
		require.ErrorIs(t, err, mergeErr)
		require.ErrorIs(t, err, emitErr)
	})

	t.Run("emit-only failure still returned", func(t *testing.T) {
		okMerge := &aggregator{
			logger:       log.NewTestLogger(),
			eventEmitter: emitter,
			shard:        service.Shard{Name: "test-join"},
			merger: &MockXferMerging{
				merged: partial,
			},
		}
		err := okMerge.withEachFile(time.Now())
		require.Error(t, err)
		require.ErrorContains(t, err, "kafka produce failed")
		require.NotContains(t, err.Error(), "unmapped input files")
		require.ErrorIs(t, err, emitErr)
	})
}

func TestEmitFilesUploaded_NilEmitter(t *testing.T) {
	xfagg := &aggregator{
		logger: log.NewTestLogger(),
		shard:  service.Shard{Name: "test-nil-emitter"},
	}
	err := xfagg.emitFilesUploaded(context.Background(), mergedFiles{
		{
			InputFilepaths:   []string{"a.ach"},
			UploadedFilename: "out.ach",
			Shard:            "test-nil-emitter",
		},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "missing event emitter")
}

func TestEmitFilesUploaded_PartialSendErrors(t *testing.T) {
	var calls atomic.Int32
	emitter := &failingNthEmitter{
		failOn: 2,
		calls:  &calls,
	}
	xfagg := &aggregator{
		logger:       log.NewTestLogger(),
		eventEmitter: emitter,
		shard:        service.Shard{Name: "test-partial"},
	}
	err := xfagg.emitFilesUploaded(context.Background(), mergedFiles{
		{
			InputFilepaths:   []string{"a.ach", "b.ach", "c.ach"},
			UploadedFilename: "out.ach",
			Shard:            "test-partial",
		},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "nth send failed")
	require.Equal(t, int32(3), calls.Load(), "every input must still be attempted")
}

type failingNthEmitter struct {
	failOn int32
	calls  *atomic.Int32
}

func (e *failingNthEmitter) Send(context.Context, models.Event) error {
	n := e.calls.Add(1)
	if n == e.failOn {
		return errors.New("nth send failed")
	}
	return nil
}
