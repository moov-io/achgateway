// generated-from:f3f35f4002746aa851a730b299373b812f8173fc53182b8fb17f63e8fd427fdd DO NOT REMOVE, DO UPDATE

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

package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/consul"
	"github.com/moov-io/achgateway/internal/dbtest"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironment_Startup(t *testing.T) {
	a := assert.New(t)

	conf := dbtest.CreateTestDatabase(t, dbtest.LocalDatabaseConfig())
	db := dbtest.LoadDatabase(t, conf)
	require.NoError(t, db.Ping())

	env := &Environment{
		Logger: log.NewDefaultLogger(),
		Config: &service.Config{
			Database: conf,
			Inbound: service.Inbound{
				InMem: &service.InMemory{
					URL: "mem://achgateway",
				},
			},
			Consul: &consul.Config{
				Address:             "127.0.0.1:8500",
				Scheme:              "http",
				SessionPath:         "achgateway/test/",
				Tags:                []string{"test1"},
				HealthCheckInterval: 10 * time.Second,
			},
		},
	}

	env, err := NewEnvironment(env)
	a.Nil(err)

	t.Cleanup(env.Shutdown)
}

type TestEnvironment struct {
	T          *testing.T
	Assert     *require.Assertions
	StaticTime stime.StaticTimeService

	Environment
}

func NewTestEnvironment(t *testing.T, router *mux.Router) *TestEnvironment {
	testEnv := &TestEnvironment{}

	testEnv.T = t
	testEnv.PublicRouter = router
	testEnv.Assert = require.New(t)
	testEnv.Logger = log.NewDefaultLogger()
	testEnv.StaticTime = stime.NewStaticTimeService()
	testEnv.TimeService = testEnv.StaticTime

	cfg, err := LoadConfig(testEnv.Logger)
	if err != nil {
		t.Fatal(err)
	}
	testEnv.Config = cfg

	cfg.Database = dbtest.CreateTestDatabase(t, dbtest.LocalDatabaseConfig())

	_, err = NewEnvironment(&testEnv.Environment)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(testEnv.Shutdown)

	return testEnv
}

func TestEnvironmentTesting(t *testing.T) {
	r := mux.NewRouter()
	env := NewTestEnvironment(t, r)
	t.Cleanup(env.Shutdown)
}

func (s TestEnvironment) MakeRequest(method string, target string, body interface{}) *http.Request {
	jsonBody := bytes.Buffer{}
	if body != nil {
		json.NewEncoder(&jsonBody).Encode(body)
	}

	return httptest.NewRequest(method, target, &jsonBody)
}

func (s TestEnvironment) MakeCall(req *http.Request, body interface{}) *http.Response {
	rec := httptest.NewRecorder()
	s.PublicRouter.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()

	if body != nil {
		json.NewDecoder(res.Body).Decode(&body)
	}

	return res
}
