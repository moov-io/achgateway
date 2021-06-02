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

	"github.com/moov-io/achgateway/internal/schedule"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"gocloud.dev/pubsub"
)

func Start(ctx context.Context, logger log.Logger, cfg *service.Config, httpFiles, streamFiles *pubsub.Subscription) error {
	xfagg := &aggregator{
		logger:      logger,
		cfg:         cfg,
		httpFiles:   httpFiles,
		streamFiles: streamFiles,
	}

	cutoffs, err := schedule.ForCutoffTimes("America/New_York", []string{"12:30"})
	if err != nil {
		return fmt.Errorf("error creating cutoffs: %v", err)
	}

	go xfagg.Start(ctx, cutoffs)

	return nil
}

type aggregator struct {
	logger log.Logger
	cfg    *service.Config

	httpFiles   *pubsub.Subscription
	streamFiles *pubsub.Subscription
}

func (xfagg *aggregator) Start(ctx context.Context, cutoffs *schedule.CutoffTimes) {
	for {
		select {
		case tt := <-cutoffs.C:
			// cutoff time
			fmt.Printf("tt=%#v\n", tt)

		case waiter := <-cutoffs.C: // TODO(adam): manual cutoff trigger
			fmt.Printf("waiter=%#v\n", waiter)

		case err := <-xfagg.await(ctx, xfagg.httpFiles): // TODO
			fmt.Printf("http err=%#v\n", err)

		case err := <-xfagg.await(ctx, xfagg.streamFiles): // TODO
			fmt.Printf("stream err=%#v\n", err)

		case <-ctx.Done():
			cutoffs.Stop()
			xfagg.Shutdown()
			return
		}
	}
}

func (xfagg *aggregator) Shutdown() {
	xfagg.logger.Log("shutting down xfer aggregation")

	// if xfagg.auditStorage != nil {
	// 	xfagg.auditStorage.Close()
	// }

	if err := xfagg.httpFiles.Shutdown(context.Background()); err != nil {
		xfagg.logger.LogErrorf("problem shutting down http file subscription: %v", err)
	}
	if err := xfagg.streamFiles.Shutdown(context.Background()); err != nil {
		xfagg.logger.LogErrorf("problem shutting down stream file subscription: %v", err)
	}
}

func (xfagg *aggregator) await(ctx context.Context, sub *pubsub.Subscription) chan error {
	out := make(chan error, 1)
	if sub == nil {
		return out
	}
	go func() {
		msg, err := sub.Receive(ctx)
		if err != nil {
			xfagg.logger.LogErrorf("ERROR receiving message: %v", err)
		}
		if msg != nil {
			xfagg.logger.Logf("begin handle received message of %d bytes", len(msg.Body))
			// err = handleMessage(xfagg.logger, xfagg.merger, msg)
			// if err != nil {
			xfagg.logger.Error().LogErrorf("end handle received message: %v", err)
			// } else {
			xfagg.logger.Log("end handle received message")
			// }
			// out <- err
			out <- nil // TODO(adam): impl handleMessage(..)
		} else {
			xfagg.logger.Log("nil message received")
		}
	}()
	return out
}

// func handleMessage() {}
