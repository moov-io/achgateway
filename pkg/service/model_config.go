// generated-from:7874fd1533ee5410fb8441565e6eb94925fc96dcdba4c71d993b84a317f4d756 DO NOT REMOVE, DO UPDATE

package service

import (
    "github.com/moov-io/base/database"
    "github.com/moovfinancial/go-zero-trust/pkg/middleware"
)

type GlobalConfig struct {
    ACH Conductor Config
}

// Config defines all the configuration for the app
type Config struct {
    Servers  ServerConfig
    Clients  *ClientConfig
    Database database.DatabaseConfig
    Gateway  middleware.ZeroTrustConfig
}

// ServerConfig - Groups all the http configs for the servers and ports that get opened.
type ServerConfig struct {
    Public HTTPConfig
    Admin  HTTPConfig
}

// HTTPConfig configuration for running an http server
type HTTPConfig struct {
    Bind BindAddress
}

// BindAddress specifies where the http server should bind to.
type BindAddress struct {
    Address string
}
