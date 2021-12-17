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

package shards

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
	"net/http"
)

func NewShardMappingController(logger log.Logger, service ShardMappingService) *ShardMappingController {
	return &ShardMappingController{
		logger:  logger,
		service: service,
	}
}

type ShardMappingController struct {
	logger  log.Logger
	service ShardMappingService
}

func (c *ShardMappingController) AppendRoutes(router *mux.Router) *mux.Router {
	router.
		Name("ShardMapping.create").
		Methods("POST").
		Path("/shard_mappings").
		HandlerFunc(c.Create)

	router.
		Name("ShardMapping.get").
		Methods("GET").
		Path("/shard_mappings/{shardKey}").
		HandlerFunc(c.Get)

	router.
		Name("ShardMapping.list").
		Methods("GET").
		Path("/shard_mappings").
		HandlerFunc(c.List)

	return router
}

func (c *ShardMappingController) Create(w http.ResponseWriter, r *http.Request) {
	body := service.ShardMapping{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := c.service.Create(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (c *ShardMappingController) Get(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	shardKey := params["shardKey"]

	shard, err := c.service.Lookup(shardKey)
	if err != nil {
		c.logger.LogErrorf("shard mapping not found by shard mapping %s: %v", shardKey, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonResponseStatus(w, http.StatusOK, shard)
}

func (c *ShardMappingController) List(w http.ResponseWriter, r *http.Request) {
	result, err := c.service.List()
	if err != nil {
		c.logger.LogErrorf("list shard mappings: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonResponseStatus(w, http.StatusOK, result)
}

func jsonResponseStatus(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	e := json.NewEncoder(w)
	e.Encode(value)
}
