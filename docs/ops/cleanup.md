---
layout: page
title: File Cleanup
---

# File Cleanup

ACHGateway v0.34.0 added an optional cleanup routine that automatically removes old processed files from the storage directory. This helps prevent file descriptor exhaustion and disk space issues in long-running deployments.

## Overview

When ACH files are processed at cutoff times, they are moved from the `mergable/<shard-name>/` directory to isolated directories with timestamps (e.g., `<shard-name>-20240115-143000/`). After successful upload to the ODFI and audit storage, these directories remain on disk indefinitely by default.

The cleanup feature periodically scans for these isolated directories and removes those that:
1. Are older than the configured retention duration
2. Have been successfully processed (contain an `uploaded/` subdirectory)

## Configuration

The cleanup feature is configured per shard in the `Upload.merging.cleanup` section:

```yaml
Upload:
  merging:
    directory: "./storage/"
    cleanup:
      enabled: true                    # Enable/disable cleanup
      retentionDuration: "24h"         # How long to keep files after processing
      checkInterval: "1h"              # How often to run cleanup
```

### Configuration Options

- **enabled**: Boolean flag to enable or disable the cleanup feature (default: `false`)
- **retentionDuration**: Duration string specifying how long to retain processed files (e.g., "24h", "7d", "168h")
- **checkInterval**: Duration string specifying how often to run the cleanup process (e.g., "1h", "30m")

### Example Configurations

#### Basic cleanup - 24 hour retention
```yaml
Upload:
  merging:
    cleanup:
      enabled: true
      retentionDuration: "24h"
      checkInterval: "1h"
```

#### Aggressive cleanup - 1 hour retention
```yaml
Upload:
  merging:
    cleanup:
      enabled: true
      retentionDuration: "1h"
      checkInterval: "15m"
```

#### Weekly cleanup - 7 day retention
```yaml
Upload:
  merging:
    cleanup:
      enabled: true
      retentionDuration: "168h"  # 7 days
      checkInterval: "24h"       # Daily check
```

## Safety Features

The cleanup service includes several safety features to prevent accidental data loss:

1. **Pattern Matching**: Only directories matching the pattern `<shard-name>-YYYYMMDD-HHMMSS` are considered for cleanup
2. **Upload Verification**: Directories are only deleted if they contain an `uploaded/` subdirectory with files, indicating successful processing
3. **Age Check**: Files must be older than the retention duration based on the timestamp in the directory name
4. **Active Directory Protection**: The `mergable/` directories used for active processing are never deleted

## Monitoring

The cleanup service exposes Prometheus metrics for monitoring:

- `achgateway_cleanup_runs_total`: Total number of cleanup runs executed (labeled by shard and status)
- `achgateway_cleanup_directories_deleted_total`: Total number of directories deleted (labeled by shard)
- `achgateway_cleanup_errors_total`: Total number of errors during cleanup (labeled by shard and error type)

## Logging

The cleanup service logs its activities at various levels:

- **INFO**: Service start/stop, cleanup run summaries, directory deletions
- **DEBUG**: Individual directory checks
- **WARN**: Non-fatal errors during directory checks
- **ERROR**: Fatal errors preventing cleanup

Example log entries:
```
INFO starting cleanup service shard=production checkInterval=1h retentionDuration=24h
INFO starting cleanup run shard=production
INFO deleting directory shard=production directory=production-20240114-143000
INFO completed cleanup run shard=production deleted=5 errors=0 duration=1.2s
```
