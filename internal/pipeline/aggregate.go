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

	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/schedule"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
)

type aggregator struct {
	logger log.Logger
	cfg    service.Shard
}

func newAggregator(logger log.Logger, cfg service.Shard) (*aggregator, error) {
	// cutoffs, err := schedule.ForCutoffTimes("America/New_York", []string{"12:30"})
	// if err != nil {
	// 	return fmt.Errorf("error creating cutoffs: %v", err)
	// }

	return &aggregator{
		logger: logger,
		cfg:    cfg,
	}, nil
}

func (xfagg *aggregator) Start(ctx context.Context, cutoffs *schedule.CutoffTimes) {
	for {
		select {
		case tt := <-cutoffs.C:
			// cutoff time
			fmt.Printf("tt=%#v\n", tt)

		case waiter := <-cutoffs.C: // TODO(adam): manual cutoff trigger
			fmt.Printf("waiter=%#v\n", waiter)

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
}

func (xfagg *aggregator) acceptFile(msg incoming.ACHFile) error {
	fmt.Printf("aggregator.acceptFile=%#v\n", msg)
	return nil
}
