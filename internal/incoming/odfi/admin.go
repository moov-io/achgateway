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
	"net/http"

	"github.com/moov-io/base/admin"
	moovhttp "github.com/moov-io/base/http"
)

func (s *PeriodicScheduler) RegisterRoutes(svc *admin.Server) {
	svc.AddHandler("/trigger-inbound", s.triggerInboundProcessing())
}

type manuallyTriggeredInbound struct {
	C chan error
}

func (s *PeriodicScheduler) triggerInboundProcessing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// send off the manual request
		waiter := manuallyTriggeredInbound{
			C: make(chan error, 1),
		}
		s.inboundTrigger <- waiter

		if err := <-waiter.C; err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			moovhttp.Problem(w, err)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}
