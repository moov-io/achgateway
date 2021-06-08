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
	"testing"

	"github.com/moov-io/achgateway/internal/dbtest"
	"github.com/moov-io/base"

	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	conf := dbtest.CreateTestDatabase(t, dbtest.LocalDatabaseConfig())
	db := dbtest.LoadDatabase(t, conf)
	require.NoError(t, db.Ping())

	shardKey := base.ID()
	shardName := "ftp-live"

	repo := NewRepository(db, nil)
	rr, ok := repo.(*sqlRepository)
	require.True(t, ok)

	err := rr.write(shardKey, shardName)
	require.NoError(t, err)

	found, err := repo.Lookup(shardKey)
	require.NoError(t, err)
	require.Equal(t, shardName, found)
}
