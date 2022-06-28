<!-- generated-from:b4257671217b641674dfb87054f6c520d6d47649f1a0a729df08e372be1133d5 DO NOT REMOVE, DO UPDATE -->
# ACH Gateway
**Purpose** | **[Configuration](CONFIGURATION.md)** | **[Running](RUNNING.md)**

---

## Goals

achgateway is an automated ACH gateway for uploading and downloading Nacha formatted files to FTP/SFTP servers.
This is a typical use case with an Originating Depository Financial Institution (ODFI) to make ACH payments.
The gateway accepts valid Nacha files across multiple interfaces and will optimize them to upload.

Several other features of achgateway include:

- Extensible submission of ACH files (and partial requests) for upload at cutoff times
- Merging multiple pending files for optimized cutoff time submission
- Custom filename templating on uploaded files
- Audit storage of uploaded and downloaded files
- Notifications on successful file upload or errors
   - Slack, PagerDuty, Emails, etc

## Non-Goals

- Nacha compliant limit analysis, transaction authorization, settlement availability, risk calculations, and other business specifics.

## Use-Cases

### ACH uploads as a service

achgateway can be used with multiple ODFI's or a desire to separate ACH uploads. The shard key that is included
on every submitted file allows for both of these usecases. Shards are designed to be mixed and used across reused
across multiple uploaders.

### micro-deposits

Validating accounts is often done with small credits submitted to an account. The experience can be improved
by originating same-day batches so the amounts are ready quickly. To implement this submission of ACH files with a
shard key of `micro-deposit` could be used and configured to upload at the end of every day. This will attempt to
minimize the files submitted along with their cost and performance.

## Usage

Follow our [getting started guide](https://github.com/moov-io/achgateway#getting-started) on the project readme to start uploading files.

### Leader Election

When an instance of achgateway receives a new `ACHFile` it will attempt a write into consul.
(See [part 4 of this article](https://clivern.com/leader-election-with-consul-and-golang/))

Writing to a path such as `/achgateway/shards/:key` is unique and offers this election capability.
Periodic refreshing of this lock is required so instance crashing is discovered after expiration.

If this write fails a read can be performed to discover the current leader.

### Watching

Nodes can be watched by non-leaders which allows them to be aware of instance crashing/shutdown and to
attempt self-election. Aggressive elections and watching should maintain an active leader who can upload files.

## Local Storage

Since every instance of achgateway consumes all files they will persist them to a local filesystem. This functions
as a "read replica" of all ACH files and allows them to take over in the event of a failed instance/leader.

## Uploading

The ACH specification describes "cutoff times" as fixed timestamps for when files must be uploaded by. This allows our
systems to work ahead of time and act as a real-time system for outside processes.

When a cutoff time is triggered there are several steps to be performed for each shard key.

1. If self-elected leader
   1. Merge pending files (inside `storage/merging/:key/*.ach`) that do not contain a `*.canceled` file.
      1. With moov-io/ach's `MergeFiles(...)` function (and optional `ach.Conditions` for max dollar amounts in a file, etc)
   1. Optionally `FlattenBatches()` on files and encrypt file contents (e.g. GPG)
   1. Render filename from template, prepare output formatting
   1. Save file to `uploaded/*.ach` inside of our `storage/merging/:key/` directory
   1. Save file to audittrail storage
   1. **Upload ACH file** to remote server
   1. Notify via Slack, PD, email, etc
   1. Future: Publish event for other services to consume

## File Merging

ACH transfers are merged (grouped) according to their file header values using [`ach.MergeFiles`](https://godoc.org/github.com/moov-io/ach#MergeFiles).
EntryDetail records are not modified as part of the merging process. Merging is done primarily to reduce the fees charged by your ODFI or The Federal Reserve.

### Uploads of Merged ACH Files

ACH files which are uploaded to another FI primarily use FTP(s) ([File Transport Protocol](https://en.wikipedia.org/wiki/File_Transfer_Protocol) with TLS) or
SFTP ([SSH File Transfer Protocol](https://en.wikipedia.org/wiki/SSH_File_Transfer_Protocol)) and follow a filename pattern like: `YYYYMMDD-ABA.ach` (example: `20181222-301234567.ach`).
The configuration file determines how achgateway uploads and transforms the files.

### Filename templates

achgateway supports templated naming of ACH files prior to their upload. This is helpful for ODFI's which require specific naming of uploaded files.
Templates use Go's [`text/template` syntax](https://golang.org/pkg/text/template/) and are validated when achgateway starts or changed via admin endpoints.

Example:

```
{{ .ShardName }}-{{ date "20060102" }}-{{ .Index }}.ach{{ if .GPG }}.gpg{{ end }}
```

The following fields are passed to templates giving them data to build a filename from:

- `ShardName`: string of the shard performing an upload
- `GPG`: boolean
- `Index`: integer

Also, several functions are available (in addition to Go's standard template functions)

- `date` Takes a Go [`Time` format](https://golang.org/pkg/time/#Time.Format) and returns the formatted string
- `env` Takes an environment variable name and returns the value from `os.Getenv`.
- `lower` and `upper` convert a string into lowercase or uppercase

Note: By default filenames have sequence numbers which are incremented by achgateway and are assumed to be in a specific format.
It is currently (as of 2019-10-14) undefined behavior that happens to incremented sequence numbers when filenames are in a different format.
Please open an issue if you run into problems here.

### Notifications

achgateway supports multiple notification options on each `Shard`. These will be pushed out on each file upload.

#### Email

Example:

```
Sharding:
  Shards:
  - id: "production"
    notifications:
      email:
      - id: "production"
        from: "noreply@company.net"
        to:
          - "ach@bank.com"
        companyName: "Acme Corp"
```

#### PagerDuty

Example:

```
Sharding:
  Shards:
  - id: "production"
    notifications:
      pagerduty:
      - id: "production"
        apiKey: "..."
        from: "..."
        serviceKey: "..."
```

#### Slack

Example:

```
Sharding:
  Shards:
  - id: "production"
    notifications:
      slack:
      - id: "production"
        webhookURL: "https://hooks.slack.com/services/..."
```

### IP Whitelisting

When achgateway uploads an ACH file to the ODFI server it can verify the remote server's hostname resolves to a whitelisted IP or CIDR range.
This supports certain network controls to prevent DNS poisoning or misconfigured routing.

Setting an `UploadAgent`'s `AllowedIPs` property can be done with values like: `35.211.43.9` (specific IP address), `10.4.0.0/16` (CIDR range), `10.1.0.12,10.3.0.0/16` (Multiple values)

### SFTP Host and Client Key Verification

achgateway can verify the remote SFTP server's host key prior to uploading files and it can have a client key provided. Both methods assist in
authenticating achgateway and the remote server prior to any file uploads.

**Public Key** (SSH Authorized key format)

```
SFTP Config: HostPublicKey
Format: ssh-rsa AAAAB...wwW95ttP3pdwb7Z computer-hostname
```

**Private Key** (PKCS#8)

```
SFTP Config: ClientPrivateKey

Format:
-----BEGIN RSA PRIVATE KEY-----
...
33QwOLPLAkEA0NNUb+z4ebVVHyvSwF5jhfJxigim+s49KuzJ1+A2RaSApGyBZiwS
...
-----END RSA PRIVATE KEY-----
```

Note: Public and Private keys can be encoded with base64 from the following formats or kept as-is. We expect Go's `base64.StdEncoding` encoding (not base64 URL encoding).

## AuditTrail

Part of Nacha's guidelines and operational best practices is to save ACH files we send off for a period of time. This allows us to
investigate customer issues and calculate analytics on those files. achgateway stores these files in an S3 compatible bucket
and encrypts the files with a GPG key.

Example GCS storage location: `gcs://achgateway-audittrail/files/2021-05-12/:shard-key:/*.ach`

## Upload Queue

Currently the input into achgateway is a pre-built ACH file that can be uploaded on its own.
This allows achgateway to optimize multiple groupable files for upload. The first example of this
is the shard key associated to every file.

achgateway can operate multiple input vectors which are merged into a singular Queue. This allows
an HTTP endpoint, kafka consumer, and other inputs.

The following messages are produced out to the Queue. Read [the `pkg/models` package](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models)
for more information on events.

```go
type QueueACHFile struct {
	FileID   string    `json:"fileID"`
	ShardKey string    `json:"shardKey"`
	File     *ach.File `json:"file"`
}
```

```go
type CancelACHFile struct {
	FileID   string `json:"id"`
	ShardKey string `json:"shardKey"`
}
```

#### HTTP Queue

An HTTP server listening on the following endpoint.

```
POST /shards/:key/files/:fileID
```

- Content-Type: `text/plain` (default)
   - Body: Nacha format
- Content-Type: `application/json`
   - Body: moov-io/ach JSON format

## Shard Mappings

Shard mapping endpoints are exposed for persisting shard mappings to and retrieving shard mappings from the database, which map a shard key to a configured shard name

#### `POST /shard_mappings`

```
POST /shard_mappings
```

Create shard mappings

**Example Request Body / Payload**

```json
{
  "shard_key": "53ce45d6-aa44-4da8-8ebb-b3daf8c1886d",
  "ach_company_id": "testing"
}
```

**Response Codes:**
- 201 - Created - The request to create a resource was successful
- 400 - Bad Request - The request payload was not serializable
- 500 - Internal Server error - Unexpected error in server, no response other than error code

#### `GET /shard_mappings`

Get shard mappings list

**Example Response Body**

```json
[
  {
    "shard_key": "53ce45d6-aa44-4da8-8ebb-b3daf8c1886d",
    "shard_name": "testing"
  },
  {
    "shard_key": "55f177da-c389-42b1-87a2-5d6a14685690",
    "shard_name": "live"
  }
]
```

**Response Codes:**
- 200 - Success - The request to get a resource was successful
- 500 - Internal Server error - Unexpected error in server, no response other than error code

### `GET /shard_mappings/{shardKey}`

Get shard by shard key

**Example Response Body**

```json
{
  "shard_key": "53ce45d6-aa44-4da8-8ebb-b3daf8c1886d",
  "shard_name": "testing"
}
```

**Response Codes:**
- 200 - Success - The request to get a resource was successful
- 404 - Not Found - The requested resource was not found
- 500 - Internal Server error - Unexpected error in server, no response other than error code

#### Kafka Queue

Consuming the `ACHFile` and `CancelACHFile` messages in JSON (or protobuf) and publishing

## Admin

### Flushing ACH Files

There is an endpoint to initiate cutoff processing as if a window has approached. This involves merging transfers into files, upload attempts, along with inbound file download processing.

```
$ curl -XPUT http://localhost:9092/trigger-cutoff
// check for errors, or '200 OK'
```

### Processing ODFI Files

There is an endpoint to initiate processing of ODFI files which could be incoming transfers, returned files, corrected files, and pre-notifications.

```
$ curl -XPUT http://localhost:9092/trigger-inbound
// check for errors, or '200 OK'
```



## Getting Help

 channel | info
 ------- | -------
[Project Documentation](https://github.com/moov-io/achgateway/tree/master/docs/) | Our project documentation is available online.
Twitter [@moov](https://twitter.com/moov)	| You can follow Moov.io's Twitter feed to get updates on our project(s). You can also tweet us questions or just share blogs or stories.
[GitHub Issue](https://github.com/moov-io/achgateway/issues) | If you are able to reproduce a problem please open a GitHub Issue under the specific project that caused the error.
[Moov Slack](https://slack.moov.io/) | Join our slack channel (`#ach`) to have an interactive discussion about the development of the project.

---
**[Next - Configuration](CONFIGURATION.md)**
