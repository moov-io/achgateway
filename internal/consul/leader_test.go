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

package consul

// import (
// 	"testing"

// 	"github.com/moov-io/base/log"
// 	"github.com/stretchr/testify/assert"
// )

// func TestAcquireLock(t *testing.T) {
// 	a := assert.New(t)
// 	logger := log.NewDefaultLogger()

// 	consulClient, err := NewConsulClient(logger, &Config{
// 		Address:     "http://127.0.0.1:8500",
// 		SessionPath: "achgateway/test/",
// 		Tags:        []string{"test1"},
// 	})
// 	a.Nil(err)

// 	testShard := "test"
// 	consulSessions := make(map[string]*Session)

// 	newSession, err := NewSession(logger, consulClient, testShard)
// 	a.Nil(err)
// 	consulSessions[testShard] = newSession
// 	a.IsType(&Session{}, consulSessions[testShard])

// 	err = AcquireLock(logger, consulClient, consulSessions[testShard])
// 	a.Nil(err)
// }

// func TestAcquireLockSessionExists(t *testing.T) {
// 	a := assert.New(t)
// 	logger := log.NewDefaultLogger()

// 	consulClient, err := NewConsulClient(logger, &Config{
// 		Address:     "http://127.0.0.1:8500",
// 		SessionPath: "achgateway/test/",
// 		Tags:        []string{"test1"},
// 	})
// 	a.Nil(err)

// 	testShard := "test2"
// 	consulSessions := make(map[string]*Session)

// 	newSession, err := NewSession(logger, consulClient, testShard)
// 	a.Nil(err)
// 	consulSessions[testShard] = newSession

// 	if _, exists := consulSessions[testShard]; exists {
// 		a.IsType(&Session{}, consulSessions[testShard])
// 	}

// 	err = AcquireLock(logger, consulClient, consulSessions[testShard])
// 	a.Nil(err)
// }
