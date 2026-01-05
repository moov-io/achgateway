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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/moov-io/base/log"
	"github.com/moov-io/base/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type manuallyTriggeredCutoff struct {
	C   chan error
	ctx context.Context
}

type manualCutoffBody struct {
	ShardNames []string `json:"shardNames"`
}

type shardResponses struct {
	Shards map[string]*string `json:"shards"`
}

func (ss shardResponses) hasErrors() bool {
	for _, err := range ss.Shards {
		if err != nil {
			return true
		}
	}
	return false
}

func (fr *FileReceiver) triggerManualCutoff() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var body manualCutoffBody
		json.NewDecoder(r.Body).Decode(&body)

		// Reject the request if no shards are specified
		if len(body.ShardNames) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx, span := telemetry.StartSpan(r.Context(), "api-manual-cutoff", trace.WithAttributes(
			attribute.StringSlice("achgateway.shards", body.ShardNames),
		))
		defer span.End()

		responses := shardResponses{
			Shards: make(map[string]*string),
		}

		for _, shardName := range body.ShardNames {
			xfagg, ok := fr.shardAggregators[shardName]
			if !ok {
				errString := fmt.Sprintf("unknown shard %s", shardName)
				responses.Shards[shardName] = &errString
				continue
			}

			logger := fr.logger.With(log.Fields{
				"shard": log.String(xfagg.shard.Name),
			})

			waiter, err := processManualCutoff(ctx, logger, xfagg)
			if err != nil {
				errString := err.Error()
				responses.Shards[xfagg.shard.Name] = &errString
				continue
			}
			if waiter == nil {
				logger.Info().Log("skipping manual trigger")
				continue
			}
			if err := <-waiter.C; err != nil {
				logger.Error().LogErrorf("ERROR when triggering shard: %v", err)
				xfagg.alertOnError(err)

				errString := err.Error()
				responses.Shards[xfagg.shard.Name] = &errString

			} else {
				logger.Info().Log("successful manual trigger")
				responses.Shards[xfagg.shard.Name] = nil
			}
		}

		// Write the response headers
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if responses.hasErrors() {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(responses)
	}
}

func processManualCutoff(ctx context.Context, logger log.Logger, xfagg *aggregator) (*manuallyTriggeredCutoff, error) {
	logger.Info().Log("found shard to manually trigger")

	waiter := manuallyTriggeredCutoff{
		C:   make(chan error, 1),
		ctx: ctx,
	}
	xfagg.cutoffTrigger <- waiter
	return &waiter, nil
}

func exists(names []string, shardName string) bool {
	for i := range names {
		if names[i] == shardName {
			return true
		}
	}
	return false
}

func (fr *FileReceiver) getShardNames() []string {
	shardNames := make([]string, 0, len(fr.shardAggregators))
	for _, xfagg := range fr.shardAggregators {
		shardNames = append(shardNames, xfagg.shard.Name)
	}
	return shardNames
}
