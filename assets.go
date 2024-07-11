package achgateway

import "embed"

//go:embed migrations/*.up.mysql.sql
var MySqlMigrationFS embed.FS

//go:embed migrations/*.up.spanner.sql
var SpannerMigrationFS embed.FS

//go:embed configs/config.default.yml
var ConfigFS embed.FS
