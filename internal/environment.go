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
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/incoming/odfi"
	"github.com/moov-io/achgateway/internal/incoming/stream"
	"github.com/moov-io/achgateway/internal/incoming/web"
	"github.com/moov-io/achgateway/internal/pipeline"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/base/admin"
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
	Consul         *consul.Client
	Events         events.Emitter

	PublicRouter *mux.Router
	AdminServer  *admin.Server
	Shutdown     func()

	HTTPFiles   *pubsub.Topic
	StreamFiles *pubsub.Topic
	ODFIFiles   odfi.Scheduler

	FileReceiver *pipeline.FileReceiver
}

// NewEnvironment - Generates a new default environment. Overrides can be specified via configs.
func NewEnvironment(env *Environment) (*Environment, error) {
	if env == nil {
		env = &Environment{}
	}

	env.Shutdown = func() {}

	var err error
	ctx, cancelFunc := context.WithCancel(context.Background()) //nolint:lostcancel
	defer func() {
		if err := recover(); err != nil {
			cancelFunc()
			panic(err)
		}
	}()

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
	if env.DB == nil && env.Config.Database.MySQL != nil {
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
			cancelFunc()
			close()
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

	// Setup our Consul client (if configured)
	if env.Consul == nil && env.Config.Consul != nil {
		consulClient, err := consul.NewConsulClient(env.Logger, env.Config.Consul)
		if err != nil {
			return nil, err
		}
		env.Consul = consulClient
		env.Logger.Info().Logf("created consul client for %s", env.Config.Consul.Address)

		prev := env.Shutdown
		env.Shutdown = func() {
			prev()
			env.Consul.Shutdown()
		}
	}

	// Setup our Events emitter
	if env.Events == nil && env.Config.Events != nil {
		emitter, err := events.NewEmitter(env.Logger, env.Config.Events)
		if err != nil {
			return nil, err
		}
		env.Events = emitter
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

	shardRepository := shards.NewRepository(env.DB, env.Config.Sharding.Mappings)
	fileReceiver, err := pipeline.Start(ctx, env.Logger, env.Config, env.Consul, shardRepository, httpSub, streamSub)
	if err != nil {
		return nil, fmt.Errorf("unable to create file pipeline: %v", err)
	}
	env.FileReceiver = fileReceiver

	// router
	if env.PublicRouter == nil {
		env.PublicRouter = mux.NewRouter()

		// append HTTP routes
		web.NewFilesController(env.Config.Logger, env.Config.Inbound.HTTP, httpFiles).AppendRoutes(env.PublicRouter)

		// shard mapping HTTP routes
		shardMappingService, err := shards.NewShardMappingService(stime.NewStaticTimeService(), env.Config.Logger, shardRepository)
		if err != nil {
			return nil, fmt.Errorf("unable to create shard mapping service: %v", err)
		}
		shards.NewShardMappingController(env.Config.Logger, shardMappingService).AppendRoutes(env.PublicRouter)
	}

	// Start our ODFI PeriodicScheduler
	if env.ODFIFiles == nil && env.Config.Inbound.ODFI != nil {
		cfg := env.Config.Inbound.ODFI
		processors := odfi.SetupProcessors(
			odfi.CorrectionEmitter(env.Logger, cfg.Processors.Corrections, env.Events),
			odfi.PrenoteEmitter(env.Logger, cfg.Processors.Prenotes, env.Events),
			odfi.CreditReconciliationEmitter(env.Logger, cfg.Processors.Reconciliation, env.Events),
			odfi.ReturnEmitter(env.Logger, cfg.Processors.Returns, env.Events),
		)
		odfiFiles, err := odfi.NewPeriodicScheduler(env.Logger, env.Config, env.Consul, processors)
		if err != nil {
			return nil, fmt.Errorf("problem creating odfi periodic scheduler: %v", err)
		}
		env.Logger.Info().Logf("starting ODFI periodic scheduler interval=%v", env.Config.Inbound.ODFI.Interval)
		env.ODFIFiles = odfiFiles

		// Start the scheduler in an anonymous goroutine
		go func() {
			if err := env.ODFIFiles.Start(); err != nil {
				env.Logger.Info().Logf("error with ODFI periodic scheduler: %v", err)
			}
		}()

		prev := env.Shutdown
		env.Shutdown = func() {
			prev()
			env.ODFIFiles.Shutdown()
		}
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
			logger.Fatal().LogErrorf("Error closing DB: %w", err)
		}
	}

	// Run the migrations
	if err := database.RunMigrations(logger, config); err != nil {
		return nil, shutdown, logger.Fatal().LogErrorf("Error running migrations: %w", err).Err()
	}

	logger.Info().Log("finished initializing db")

	return db, shutdown, err
}
