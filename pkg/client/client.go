// generated-from:b1c98792533d4c3379ca57f173bd058cadc51eb73e9b6d8633341c1ae1881aa0 DO NOT REMOVE, DO UPDATE

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

package client

import "net/http"

// Stubs to keep test errors from happening when bringing in the client.
// This will be overritten the first time you run the openapi-generator
type Configuration struct {
	HTTPClient *http.Client
}

func NewConfiguration() *Configuration {
	return &Configuration{}
}

type APIClient struct {
	cfg *Configuration
}

func NewAPIClient(cfg *Configuration) *APIClient {
	return &APIClient{
		cfg: NewConfiguration(),
	}
}
