## How an ACH file gets uploaded

1. Consume from Queue (Kafka, HTTP, etc)
1. Inspect Shard Key
   1. Seen before? Self-Elect as leader
   1. Otherwise, watch node
1. Persist file to local storage
   - `storage/pending/:shard-key:/*.ach`
1. On cutoff, for each self-elected shard I lead
   1. Merge files, flatten batches, encrypt, etc
   1. Save merged file
      - `storage/pending/:shard-key:/uploaded/*.ach`
   1. Persist to audittrail storage
      - `gcs://achgateway-audittrail/files/2021-05-12/:shard-key:/*.ach`
   1. Notify via Slack, PD, Email, etc

Notes:

- When watching nodes, if a watch expires we care about try and self-elect as leader
- Cancel messages come through and write `*.canceled` files

## Queue

Currently the input into achgateway is a pre-built ACH file that can be uploaded on its own.
This allows achgateway to optimize multiple groupable files for upload. The first example of this
is the shard key associated to every file.

achgateway can operate multiple input vectors which are merged into a singular Queue. This allows
an HTTP endpoint, kafka consumer, and other inputs.

The following messages are produced out of the Queue.

```go
type ACHFile struct {
	ID       string    `json:"id"`
	ShardKey string    `json:"shardKey"`
	File     *ach.File `json:"file"`
}
```

```go
type CancelACHFile struct {
	ID       string `json:"id"`
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

#### Kafka Queue

Consuming the `ACHFile` and `CancelACHFile` messages in JSON (or protobuf) and publishing

#### Multi

A `for { select { ... } }` loop over multiple Queue types.

## Sharding

achgateway is a distributed system that coordinates with other instances of itself to upload files.
As a design choice we make a few claims about each instance of achgateway:

1. Each instance will consume ALL files encountered
   - In the future we can have them consume a fraction of shard keys (or specific values) to shed load
1. Instances will attempt self-election for each shard key they encounter
   - Included heartbeats to refresh and maintain leadership
1. At cutoff times each leader will attempt uploads for its shard

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

Writing files to disk is similar to what paygate-worker does and would be done in a path like: `storage/pending/:shard-key:/*.ach`

## Uploading

The ACH specification describes "cutoff times" as fixed timestamps for when files must be uploaded by. This allows our
systems to work ahead of time and act as a real-time system for outside processes.

When a cutoff time is tirggered there are several steps to be performed for each shard key.

1. If self-elected leader
   1. Merge pending files (inside `storage/pending/:key/*.ach`) that do not contain a `*.canceled` file.
      1. With moov-io/ach's `MergeFiles(...)` function
   1. Optionally `FlattenBatches()` on files and encrypt file contents (e.g. GPG)
   1. Render filename from template, prepare output formatting
   1. Save file to `uploaded/*.ach` inside of our `storage/pending` directory
   1. Save file to audittrail storage
   1. **Upload ACH file** to remote server
   1. Notify via Slack, PD, email, etc
   1. Future: Publish event for other services to consume

## AuditTrail

Part of Nacha's guidelines and operational best practices is to save ACH files we send off for a period of time. This allows us to
investigate customer issues and calculate analytics on those files. achgateway stores these files in an S3 compatiable bucket
and encrypts the files with a GPG key.

Example GCS storage location: `gcs://achgateway-audittrail/files/2021-05-12/:shard-key:/*.ach`
