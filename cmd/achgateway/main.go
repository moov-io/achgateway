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
	"flag"
	"fmt"
	"os"

	"github.com/moov-io/achgateway"
	"github.com/moov-io/achgateway/internal"
	"github.com/moov-io/achgateway/internal/kafka"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
)

func main() {
	fs := flag.NewFlagSet("achgateway", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: achgateway [options]\n\n")
		fmt.Fprintf(os.Stderr, "ACH Gateway queues ACH files and uploads them to an ODFI at cutoff times.\n")
		fmt.Fprintf(os.Stderr, "Configuration is loaded from configs/config.default.yml and optional files\n")
		fmt.Fprintf(os.Stderr, "specified by APP_CONFIG and APP_CONFIG_SECRETS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(os.Args[1:]); err != nil {
		// flag already printed the error and usage for -h / parse errors
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}
	if *showVersion {
		fmt.Println(achgateway.Version)
		os.Exit(0)
	}

	env := &internal.Environment{
		Logger: log.NewDefaultLogger().Set("app", log.String("achgateway")).Set("version", log.String(achgateway.Version)),
	}

	kafka.EnableSaramaDebugLogging(env.Logger)

	env, err := internal.NewEnvironment(env)
	if err != nil {
		env.Logger.Fatal().LogErrorf("Error loading up environment: %v", err)
		os.Exit(1)
	}
	defer env.Shutdown()

	termListener := service.NewTerminationListener()

	stopServers := env.RunServers(termListener)
	defer stopServers()

	if env.AdminServer != nil && env.ODFIFiles != nil {
		env.ODFIFiles.RegisterRoutes(env.AdminServer)
	}

	service.AwaitTermination(env.Logger, termListener)
}
