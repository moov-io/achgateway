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
	"encoding/json"
	"net/http"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
)

type manuallyTriggeredCutoff struct {
	C chan error
}

type manualCutoffBody struct {
	ShardNames []string `json:"shardNames"`
}

func (fr *FileReceiver) triggerManualCutoff() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var body manualCutoffBody
		json.NewDecoder(r.Body).Decode(&body)

		var errs struct {
			Errors []string `json:"errors"`
		}

		for _, xfagg := range fr.shardAggregators {
			logger := fr.logger.With(log.Fields{
				"shard": log.String(xfagg.shard.Name),
			})

			waiter, err := processManualCutoff(logger, body.ShardNames, xfagg.shard, xfagg)
			if err != nil {
				errs.Errors = append(errs.Errors, err.Error())
				continue
			}
			if waiter == nil {
				logger.Info().Log("skipping manual trigger")
				continue
			}
			if err := <-waiter.C; err != nil {
				logger.Error().LogErrorf("ERROR when triggering shard: %v", err)
				xfagg.alertOnError(err)
				errs.Errors = append(errs.Errors, err.Error())
			} else {
				logger.Info().Log("successful manual trigger")
			}
		}

		if len(errs.Errors) > 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(errs)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func processManualCutoff(logger log.Logger, shardNames []string, shard service.Shard, xfagg *aggregator) (*manuallyTriggeredCutoff, error) {
	if !exists(shardNames, shard.Name) {
		return nil, nil
	}

	logger.Info().Log("found shard to manually trigger")

	waiter := manuallyTriggeredCutoff{
		C: make(chan error, 1),
	}
	xfagg.cutoffTrigger <- waiter
	return &waiter, nil
}

func exists(names []string, shardName string) bool {
	if len(names) == 0 {
		return true
	}
	for i := range names {
		if names[i] == shardName {
			return true
		}
	}
	return false
}
