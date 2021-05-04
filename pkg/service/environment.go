// generated-from:994cb1807d5be296a4c90c1ea79a7e538823ce94d7dbd4e90f0ec5b01abf1e63 DO NOT REMOVE, DO UPDATE

package service

import (
	"net/http"
    "context"
    "database/sql"
	"time"

    "github.com/gorilla/mux"
    "github.com/moov-io/base/config"
    "github.com/moov-io/base/database"
    "github.com/moov-io/base/log"
    "github.com/moov-io/base/stime"
    "github.com/moovfinancial/go-zero-trust/pkg/middleware"

    _ "github.com/moov-io/ach-conductor"
)

// Environment - Contains everything thats been instantiated for this service.
type Environment struct {
    Logger              log.Logger
    Config              *Config
    TimeService         stime.TimeService
    ZeroTrustMiddleware mux.MiddlewareFunc
    DB                  *sql.DB
    InternalClient      *http.Client

    
    PublicRouter *mux.Router
    Shutdown     func()
}

// NewEnvironment - Generates a new default environment. Overrides can be specified via configs.
func NewEnvironment(env *Environment) (*Environment, error) {
    if env == nil {
        env = &Environment{}
    }

    env.Shutdown = func(){}

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
        }
    }

    if env.InternalClient == nil {
        env.InternalClient = NewInternalClient(env.Logger, env.Config.Clients, "internal-client")
    }

    if env.TimeService == nil {
        env.TimeService = stime.NewSystemTimeService()
    }

    if env.ZeroTrustMiddleware == nil {
        // auth middleware for the tokens coming from the gateway
        gatewayMiddleware, err := middleware.NewServerFromConfig(env.Logger, env.TimeService, env.Config.Gateway)
        if err != nil {
            return nil, env.Logger.Fatal().LogErrorf("failed to startup Gateway middleware: %w", err).Err()
        }
        env.ZeroTrustMiddleware = gatewayMiddleware.Handler
    }

    // router
    if env.PublicRouter == nil {
        env.PublicRouter = mux.NewRouter()

        // @TODO add controller connections here
    }

    env.PublicRouter.Use(env.ZeroTrustMiddleware)

    return env, nil
}

func LoadConfig(logger log.Logger) (*Config, error) {
    configService := config.NewService(logger)

    global := &GlobalConfig{}
    if err := configService.Load(global); err != nil {
        return nil, err
    }

    cfg := &global.ACH Conductor

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
