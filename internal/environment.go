// generated-from:1157fc1ef166852fadd506762471145d5f05e75aed87bdbc6f88e67b8e4479cc DO NOT REMOVE, DO UPDATE

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
	"database/sql"
	"fmt"
	"net/http"
	"time"

	_ "github.com/moov-io/achgateway"
	"github.com/moov-io/achgateway/internal/consul"
	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/incoming/web"
	"github.com/moov-io/achgateway/internal/pipeline"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/config"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"

	"github.com/gorilla/mux"
	"gocloud.dev/pubsub"
)

// Environment - Contains everything thats been instantiated for this service.
type Environment struct {
	Logger         log.Logger
	Config         *service.Config
	TimeService    stime.TimeService
	DB             *sql.DB
	InternalClient *http.Client
	ConsulClient   *consul.Client
	ConsulSessions map[string]*consul.Session

	PublicRouter *mux.Router
	Shutdown     func()

	HTTPFiles   *pubsub.Topic
	StreamFiles *pubsub.Topic
}

// NewEnvironment - Generates a new default environment. Overrides can be specified via configs.
func NewEnvironment(env *Environment) (*Environment, error) {
	if env == nil {
		env = &Environment{}
	}

	var err error
	ctx, cancelFunc := context.WithCancel(context.Background())

	env.Shutdown = func() {}

	if env.Logger == nil {
		env.Logger = log.NewDefaultLogger()
	}

	if env.Config == nil {
		cfg, err := LoadConfig(env.Logger)
		if err != nil {
			return nil, err
		}
		env.Config = cfg
	}
	env.Config.Logger = env.Logger

	// db setup
	if env.DB == nil {
		db, close, err := initializeDatabase(env.Logger, env.Config.Database)
		if err != nil {
			close()
			return nil, err
		}
		env.DB = db

		// Add DB closing to the Shutdown call for the Environment
		prev := env.Shutdown
		env.Shutdown = func() {
			prev()
			close()
			cancelFunc()
		}
	}

	if env.InternalClient == nil {
		env.InternalClient = service.NewInternalClient(env.Logger, env.Config.Clients, "internal-client")
	}

	if env.TimeService == nil {
		env.TimeService = stime.NewSystemTimeService()
	}

	// File publishers
	inmemConfig := &service.Config{
		Inbound: service.Inbound{
			InMem: &service.InMemory{
				URL: "mem://achgateway",
			},
		},
	}
	httpFiles, err := stream.Topic(env.Logger, inmemConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create http files: %v", err)
	}

	// router
	if env.PublicRouter == nil {
		env.PublicRouter = mux.NewRouter()

		// append HTTP routes
		web.NewFilesController(env.Config.Logger, httpFiles).AppendRoutes(env.PublicRouter)
	}

	// file pipeline
	httpSub, err := stream.Subscription(env.Logger, inmemConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create http files subscription: %v", err)
	}
	streamSub, err := stream.Subscription(env.Logger, env.Config)
	if err != nil {
		return nil, fmt.Errorf("unable to create stream files subscription: %v", err)
	}

	err = pipeline.Start(ctx, env.Logger, env.Config, httpSub, streamSub)
	if err != nil {
		return nil, fmt.Errorf("unable to create file pipeline: %v", err)
	}

	if env.ConsulClient == nil {
		consulClient, err := consul.NewConsulClient(env.Logger, &consul.Config{
			Address:                    env.Config.Consul.Address,
			Scheme:                     env.Config.Consul.Scheme,
			Tags:                       env.Config.Consul.Tags,
			HealthCheckIntervalSeconds: env.Config.Consul.HealthCheckIntervalSeconds,
		})
		if err != nil {
			return nil, err
		}

		consulSession, err := consul.NewSession(env.Logger, *consulClient, consulClient.NodeId)
		if err != nil {
			return nil, err
		}
		env.ConsulClient = consulClient
		if env.ConsulSessions == nil {
			env.ConsulSessions = map[string]*consul.Session{}
		}
		env.ConsulSessions[consulClient.NodeId] = consulSession
	}

	return env, nil
}

func LoadConfig(logger log.Logger) (*service.Config, error) {
	configService := config.NewService(logger)

	global := &service.GlobalConfig{}
	if err := configService.Load(global); err != nil {
		return nil, err
	}

	cfg := &global.ACHGateway

	return cfg, nil
}

func initializeDatabase(logger log.Logger, config database.DatabaseConfig) (*sql.DB, func(), error) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	// connect to the database and keep retrying
	db, err := database.New(ctx, logger, config)
	for i := 0; err != nil && i < 22; i++ {
		logger.Info().Log("attempting to connect to database again")
		time.Sleep(time.Second * 5)
		db, err = database.New(ctx, logger, config)
	}
	if err != nil {
		return nil, cancelFunc, logger.Fatal().LogErrorf("Error creating database: %w", err).Err()
	}

	shutdown := func() {
		logger.Info().Log("Shutting down the db")
		cancelFunc()
		if err := db.Close(); err != nil {
			logger.Fatal().LogErrorf("Error closing DB", err)
		}
	}

	// Run the migrations
	if err := database.RunMigrations(logger, config); err != nil {
		return nil, shutdown, logger.Fatal().LogErrorf("Error running migrations: %w", err).Err()
	}

	logger.Info().Log("finished initializing db")

	return db, shutdown, err
}
