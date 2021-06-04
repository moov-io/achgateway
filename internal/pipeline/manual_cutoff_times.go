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
	"fmt"
	"net/http"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/admin"
)

func (fr *FileReceiver) RegisterAdminRoutes(r *admin.Server) {
	r.AddHandler("/trigger-cutoff", fr.triggerManualCutoff())
}

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
			waiter, err := processManualCutoff(body.ShardNames, xfagg.shard, xfagg)
			if err != nil {
				errs.Errors = append(errs.Errors, err.Error())
				continue
			}
			if err := <-waiter.C; err != nil {
				xfagg.alerter.AlertError(err)
				errs.Errors = append(errs.Errors, err.Error())
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

func processManualCutoff(shardNames []string, shard service.Shard, xfagg *aggregator) (*manuallyTriggeredCutoff, error) {
	if !exists(shardNames, shard.Name) {
		return nil, fmt.Errorf("unexpected shard to process")
	}

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
