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
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"
	"github.com/pkg/errors"
)

type ShardMappingService interface {
	Create(create *service.ShardMapping) (*service.ShardMapping, error)
	List() ([]service.ShardMapping, error)
	Lookup(shardKey string) (string, error)
}

func NewShardMappingService(time stime.TimeService, logger log.Logger, repository Repository) (ShardMappingService, error) {
	return &shardMappingService{
		time:       time,
		logger:     logger,
		repository: repository,
	}, nil
}

type shardMappingService struct {
	time       stime.TimeService
	logger     log.Logger
	repository Repository
}

func (s *shardMappingService) Create(create *service.ShardMapping) (*service.ShardMapping, error) {
	if err := create.Validate(); err != nil {
		return nil, errors.Wrap(err, "validating")
	}

	created := service.ShardMapping{
		ShardKey:  create.ShardKey,
		ShardName: create.ShardName,
	}

	err := s.repository.Add(created, database.NopInTx)
	if err != nil {
		return nil, errors.Wrap(err, "adding Facilitator")
	}

	return &created, nil
}

func (s *shardMappingService) List() ([]service.ShardMapping, error) {
	return s.repository.List()
}

func (s *shardMappingService) Lookup(shardKey string) (string, error) {
	return s.repository.Lookup(shardKey)
}
