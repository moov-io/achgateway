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
	"net/http"
	"testing"

	"github.com/moov-io/achgateway/internal/service"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestAdminConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("-short flag specified")
	}

	r := mux.NewRouter()
	env := NewTestEnvironment(t, r)
	t.Cleanup(env.Shutdown)

	env.RunServers(service.NewTerminationListener())
	env.registerConfigRoute()

	req, err := http.NewRequest("GET", "http://"+env.AdminServer.BindAddr()+"/config", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	if resp.Body != nil {
		resp.Body.Close()
	}
}
