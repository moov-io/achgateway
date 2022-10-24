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

type InMemoryRepository struct {
	Shards []service.ShardMapping
}

func NewInMemoryRepository(shards ...service.ShardMapping) *InMemoryRepository {
	repo := &InMemoryRepository{}
	repo.Shards = append(repo.Shards, shards...)
	return repo
}

func (r *InMemoryRepository) Lookup(shardKey string) (string, error) {
	for i := range r.Shards {
		if r.Shards[i].ShardKey == shardKey {
			return r.Shards[i].ShardName, nil
		}
	}
	return "", fmt.Errorf("unknown shardKey=%s", shardKey)
}

func (r *InMemoryRepository) List() ([]service.ShardMapping, error) {
	list := make([]service.ShardMapping, 0, len(r.Shards))
	list = append(list, r.Shards...)
	return list, nil
}

func (r *InMemoryRepository) Add(create service.ShardMapping, _ database.RunInTx) error {
	r.Shards = append(r.Shards, create)
	return nil
}
