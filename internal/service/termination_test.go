// generated-from:1707fd7fce48bdd1cbfbbd9efcc7347ad3bdc8b6b8286d28dde59f4d919c4df0 DO NOT REMOVE, DO UPDATE

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
	"errors"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTermination(t *testing.T) {
	listener := service.NewTerminationListener()
	err := make(chan error)
	go func() {
		err <- service.AwaitTermination(log.NewTestLogger(), listener)
	}()
	listener <- errors.New("foo")

	got := <-err
	require.Error(t, got)
	assert.Contains(t, got.Error(), "Terminated: foo")
}
