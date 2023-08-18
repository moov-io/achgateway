// generated-from:5ab038b99443ce42535e7fe7fa6c5a8cdb79a918bc36f1900ae5e3165a160f55 DO NOT REMOVE, DO UPDATE

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

package dbtest

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/moov-io/base"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"
)

func LocalDatabaseConfig() database.DatabaseConfig {
	return database.DatabaseConfig{
		DatabaseName: "achgateway",
		MySQL: &database.MySQLConfig{
			Address:  "tcp(127.0.0.1:3306)",
			User:     "root",
			Password: "root",
		},
	}
}

func CreateTestDatabase(t *testing.T, config database.DatabaseConfig) database.DatabaseConfig {
	open := func() (*sql.DB, error) {
		db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@%s/", config.MySQL.User, config.MySQL.Password, config.MySQL.Address))
		if err != nil {
			return nil, err
		}

		if err := db.Ping(); err != nil {
			return nil, err
		}

		return db, nil
	}

	rootDb, err := open()
	for i := 0; err != nil && i < 22; i++ {
		time.Sleep(time.Second * 1)
		rootDb, err = open()
	}
	if err != nil {
		t.Fatal(err)
	}

	dbName := "test" + base.ID()
	_, err = rootDb.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		rootDb.Exec(fmt.Sprintf("DROP DATABASE %s", dbName))
		rootDb.Close()
	})

	config.DatabaseName = dbName

	return config
}

func LoadDatabase(t *testing.T, config database.DatabaseConfig) *sql.DB {
	l := log.NewTestLogger()
	db, err := database.New(context.Background(), l, config)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	err = database.RunMigrations(l, config)
	if err != nil {
		t.Error(err)
	}

	return db
}
