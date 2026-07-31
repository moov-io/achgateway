package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration mapping.
type Config struct {
	ACHGateway ACHGateway `yaml:"ACHGateway"`
}

// ACHGateway holds the application namespaces.
type ACHGateway struct {
	Admin     AdminConfig     `yaml:"Admin"`
	Telemetry TelemetryConfig `yaml:"Telemetry"`
	Inbound   InboundConfig   `yaml:"Inbound"`
	// Database is optional (commented out in the default YAML); use nil when not configured.
	Database *DatabaseConfig `yaml:"Database,omitempty"`
}

type AdminConfig struct {
	BindAddress string `yaml:"BindAddress"` // e.g. ":9494"
}

type TelemetryConfig struct {
	ServiceName string `yaml:"ServiceName"` // e.g. "achgateway"
}

type InboundConfig struct {
	HTTP HTTPConfig `yaml:"HTTP"`
}

type HTTPConfig struct {
	BindAddress string `yaml:"BindAddress"` // e.g. ":8484"
}

type DatabaseConfig struct {
	DatabaseName string       `yaml:"DatabaseName"`
	MySQL        *MySQLConfig `yaml:"MySQL,omitempty"`
}

type MySQLConfig struct {
	Address  string `yaml:"Address"`  // e.g. "tcp(localhost:3306)"
	User     string `yaml:"User"`     // e.g. "achgateway"
	Password string `yaml:"Password"` // e.g. "achgateway"
}

// LoadFile reads a YAML file at the given path and unmarshals it into Config.
func LoadFile(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// DefaultConfig returns a config populated with the same defaults shown in the YAML file.
func DefaultConfig() *Config {
	return &Config{
		ACHGateway: ACHGateway{
			Admin: AdminConfig{
				BindAddress: ":9494",
			},
			Telemetry: TelemetryConfig{
				ServiceName: "achgateway",
			},
			Inbound: InboundConfig{
				HTTP: HTTPConfig{
					BindAddress: ":8484",
				},
			},
			// Database remains nil by default to mirror the commented-out section.
		},
	}
}
