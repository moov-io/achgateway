// generated-from:9e0782b937278abaee17ffb9be40bb7928f6d9aeac4d35aa713f071163fd474c DO NOT REMOVE, DO UPDATE

package service

import (
	"github.com/moov-io/base/database"
)

type GlobalConfig struct {
	ACHConductor Config
}

// Config defines all the configuration for the app
type Config struct {
	Servers  ServerConfig
	Clients  *ClientConfig
	Database database.DatabaseConfig
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
