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
	"fmt"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/database"
)

type MockRepository struct {
	Shards map[string]service.ShardMapping
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		Shards: make(map[string]service.ShardMapping),
	}
}

func (r *MockRepository) Lookup(shardKey string) (string, error) {
	if shard, exists := r.Shards[shardKey]; exists {
		return shard.ShardName, nil
	}
	return "", fmt.Errorf("unknown shardKey=%s", shardKey)
}

func (r *MockRepository) List() ([]service.ShardMapping, error) {
	list := make([]service.ShardMapping, 0, len(r.Shards))
	for _, shard := range r.Shards {
		list = append(list, shard)
	}
	return list, nil
}

func (r *MockRepository) Add(create service.ShardMapping, run database.RunInTx) error {
	r.Shards[create.ShardKey] = create
	return nil
}
