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

package service_test

import (
	"testing"

	"github.com/moov-io/achgateway/internal/consul"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/test"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/assert"
)

func Test_Environment_Startup(t *testing.T) {
	a := assert.New(t)

	env := &service.Environment{
		Logger: log.NewDefaultLogger(),
		Config: &service.Config{
			Database: test.TestDatabaseConfig(),
			Consul: &consul.Config{
				Address:                    "127.0.0.1:8500",
				Scheme:                     "http",
				SessionPath:                "achgateway/test/",
				Tags:                       []string{"test1"},
				HealthCheckIntervalSeconds: 10,
			},
		},
	}

	env, err := service.NewEnvironment(env)
	a.Nil(err)

	t.Cleanup(env.Shutdown)
}
