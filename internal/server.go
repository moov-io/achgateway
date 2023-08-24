// generated-from:830f6303750b0b2de97694207e0c9ef2e7fda1b1cb2c97b8fdcea65d3b08a234 DO NOT REMOVE, DO UPDATE

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
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/log"

	_ "github.com/moov-io/achgateway"
)

// RunServers - Boots up all the servers and awaits till they are stopped.
func (env *Environment) RunServers(terminationListener chan error) func() {
	env.AdminServer = bootAdminServer(terminationListener, env.Logger, env.Config.Admin)

	// register the admin routes
	env.registerConfigRoute()
	env.FileReceiver.RegisterAdminRoutes(env.AdminServer)

	_, shutdownPublicServer := bootHTTPServer("public", env.PublicRouter, terminationListener, env.Logger, env.Config.Inbound.HTTP)

	return func() {
		env.AdminServer.Shutdown()
		shutdownPublicServer()
	}
}

func bootHTTPServer(name string, routes *mux.Router, errs chan<- error, logger log.Logger, config service.HTTPConfig) (*http.Server, func()) {
	// Create main HTTP server
	serve := &http.Server{
		Addr:    config.BindAddress,
		Handler: routes,
		TLSConfig: &tls.Config{
			InsecureSkipVerify:       false,
			PreferServerCipherSuites: true,
			MinVersion:               tls.VersionTLS12,
		},
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start main HTTP server
	go func() {
		if config.TLS.CertFile != "" && config.TLS.KeyFile != "" {
			logger.Info().Log(fmt.Sprintf("%s listening on %s for HTTPS", name, config.BindAddress))
			if err := serve.ListenAndServeTLS(config.TLS.CertFile, config.TLS.KeyFile); err != nil {
				errs <- logger.Fatal().LogErrorf("problem starting https: %w", err).Err()
			}
		} else {
			logger.Info().Log(fmt.Sprintf("%s listening on %s for HTTP", name, config.BindAddress))
			if err := serve.ListenAndServe(); err != nil {
				errs <- logger.Fatal().LogErrorf("problem starting http: %w", err).Err()
			}
		}
	}()

	shutdownServer := func() {
		if err := serve.Shutdown(context.Background()); err != nil {
			logger.Fatal().LogErrorf("shutting down: %v", err)
		}
	}

	return serve, shutdownServer
}

func bootAdminServer(errs chan<- error, logger log.Logger, config service.Admin) *admin.Server {
	adminServer, err := admin.New(admin.Opts{
		Addr: config.BindAddress,
	})
	if err != nil {
		errs <- logger.Fatal().LogErrorf("problem creating admin server: %v", err).Err()
		return nil
	}

	go func() {
		logger.Info().Log(fmt.Sprintf("listening on %s", adminServer.BindAddr()))
		if err := adminServer.Listen(); err != nil {
			errs <- logger.Fatal().LogErrorf("problem starting admin http: %w", err).Err()
		}
	}()

	return adminServer
}
