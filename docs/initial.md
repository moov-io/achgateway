# ach-conductor

## Goals

- Extensible submission of ACH files (and partial requests) for upload at cutoff times
- Merging multiple pending files for optimized cutoff time submission
- Custom filename templating on uploaded files
- Audit storage of uploaded and downloaded files
- Notifications on successful file upload or errors
   - Slack, PagerDuty, Emails, etc

## Non-Goals

- UI for viewing, editing, etc ACH files

## Steps

1. kafka consumption and
1. consul leadership
1. trivial file upload
1. benchmark entire setup (3 conductors, 3 consul nodes)
1. add ACH specifics for merge, upload, etc

## High Level Plan

Currently `paygate-worker` accepts the following messages on a kafka topic:

```go
type Xfer struct {
	TenantID string            `json:"tenantID"`
	Transfer *paygate.Transfer `json:"transfer"`
	File     *ach.File         `json:"file"`
}
```

```go
type CanceledTransfer struct {
	TenantID   string `json:"tenantID"`
	TransferID string `json:"transferID"`
}
```

From here we aggregate, merge, and upload ACH files according to cutoff times configured.
Let's start from that and have an interface to transform these kafka messages into, so we
could accept other input forms (HTTP POST, other messages, etc).

```go
type ACHFile struct {
    ID       string
    ShardKey string
    File     *ach.File
}
```

```go
type CancelACHFile struct {
    ID       string
    ShardKey string
}
```

Configuration shifts from "TenantID" over to "ShardKey" where one leader is elected prior to
upload times, but all replicas consume the files. Leave the configuration extensible so we can
throttle this replication.

Multiple instances are setup and initiate elections for a shardkey when encountered. This leader
has the responsibility for uploading files at a cutoff time.

Each instance heartbeats that leader and reports the status. There could be a prometheus metric
checking for `count(up{instance="..."}) < 1` (or checking each instance's status of heartbeating).

## Configuration

```yaml
inbound:
  http:
    bindAddress: ":8080"
  kafka:
    brokers:
      - <string>
    key: <string>
    secret: <string>
    group: <string>

shards:
  - id: "production"
    upload:
      agent: "sftp:prod"
    cutoffs:
      timezone: "America/New_York"
      windows:
        - "12:30"
    output:
      format: "nacha"
    notifications:
      email:
        - "email:prod"
      slack:
        - "slack:prod"
      pagerduty:
        - "pagerduty:prod"
    auditTrail:
      id: "audit:prod"

  - id: "testing"
    upload:
      agent: "ftp:test"
    cutoffs:
      # ...
    notifications:
      email:
        - "email:test"
      slack:
        - "slack:test"
    auditTrail:
      id: "audit:test"

  - id: "micro-deposits"
    upload:
      agent: "sftp:prod"
    cutoffs:
      timezone: "America/New_York"
      windows:
        - "16:30" # Last cutoff for the day
    output:
      format: "nacha"
    notifications:
      email:
        - "email:prod"
      slack:
        - "slack:prod"
      pagerduty:
        - "pagerduty:prod"
    auditTrail:
      id: "audit:prod"

upload:
  agents:
    - id: "sftp:prod"
      sftp:
        username: <string>
        # ...
      paths:
        outbound: <string>
        # ...
    - id: "ftp:test"
      ftp:
        username: <string>
        # ...
      paths:
        # ...

notifications:
  email:
    - id: "email:prod"
      from: noreply@moov.io
      # ...
  slack:
    - id: "slack:prod"
      webhookURL: <string>
    - id: "slack:test"
      webhookURL: <string>
  pagerduty:
    - id: "pagerduty:prod"
      apiKey: <string>

auditTrail:
  - id: "audit:prod"
    gcs:
      bucketURI: <string>
```

### Database

Inside a `shard_configs` table:

| `shard_key`      | `shard_id`       |
|------------------|------------------|
| `tenant`         | `production`     |
| `tenant1`        | `production`     |
| `moov-tenant`    | `testing`        |
| `beta-tenant`    | `testing`        |
| `micro-deposits` | `micro-deposits` |
