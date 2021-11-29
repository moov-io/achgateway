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
	"database/sql"
	"fmt"
)

type Repository interface {
	Lookup(shardKey string) (string, error)
}

func NewRepository(db *sql.DB, static map[string]string) Repository {
	if db == nil {
		return &MockRepository{Shards: static}
	}
	return &sqlRepository{db: db}
}

type sqlRepository struct {
	db *sql.DB
}

func (r *sqlRepository) Lookup(shardKey string) (string, error) {
	query := `
		SELECT
		    shard_name
		FROM
		     shard_mappings
		WHERE
		      shard_key = ? LIMIT 1;
  		`
	rows, err := r.db.Query(query, shardKey)
	if err != nil {
		return "", err
	}
	if rows.Err() != nil {
		return "", rows.Err()
	}
	defer rows.Close()

	var shardName string
	for rows.Next() {
		err := rows.Scan(&shardName)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", nil
			}
			return "", err
		}
	}
	return shardName, nil
}

func (r *sqlRepository) write(shardKey, shardName string) error {
	query := `INSERT INTO shard_mappings (shard_key, shard_name) VALUES (?, ?);`
	result, err := r.db.Exec(query, shardKey, shardName)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected < 1 {
		return fmt.Errorf("db failed to write shard mapping for shard %s-%s", shardKey, shardName)
	}
	return err
}
