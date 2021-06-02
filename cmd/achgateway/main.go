// generated-from:9d2e1a7aff438bb75e877b034d21b525c8c10efee44288edf6ce935500a9fe76 DO NOT REMOVE, DO UPDATE

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

package main

import (
	"fmt"
	"os"

	"github.com/moov-io/achgateway"
	"github.com/moov-io/achgateway/internal"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
)

func main() {
	env := &internal.Environment{
		Logger: log.NewDefaultLogger().Set("app", log.String("achgateway")).Set("version", log.String(achgateway.Version)),
	}

	env, err := internal.NewEnvironment(env)
	if err != nil {
		panic(fmt.Sprintf("ERROR: %v", err))
		// env.Logger.Fatal().LogErrorf("Error loading up environment: %v", err)
		os.Exit(1)
	}
	defer env.Shutdown()

	termListener := service.NewTerminationListener()

	stopServers := env.RunServers(termListener)
	defer stopServers()

	service.AwaitTermination(env.Logger, termListener)
}
