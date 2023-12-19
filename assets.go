package achgateway

import "embed"

//go:embed migrations/*.sql
var MigrationFS embed.FS

//go:embed configs/config.default.yml
var ConfigFS embed.FS
