// generated-from:0fb19a540a93e15a07bd0e3f1f053a620dc7d42baeef277f131e9c64d13d13d7 DO NOT REMOVE, DO UPDATE

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

package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

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
