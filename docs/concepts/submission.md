---
layout: page
title: Submitting Files
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# File Submission

ACH is a fixed-width file format used to debit or credit bank accounts. Often businesses will use this payment method to collect or distribute funds to their customers/users. The main record within an ACH file to accomplish this [EntryDetail records are created](https://moov-io.github.io/ach/file-structure/#entry-detail-record) to specify the amount, account, and description. For more details refer to Gusto's [post covering how ACH works](https://engineering.gusto.com/how-ach-works-a-developer-perspective-part-4/) from a developer point of view.

ACHGateway does not cover the business rules or Nacha requirements of creating ACH files. ACHGateway does allow files to be submitted and queued for their eventual delivery to the Federal Reserve. There are "cutoff windows" throughout a banking day to flush files from ODFI's to RDFI's (receiving financial institutions).

<a href="../../images/OSS_File_Submission.png"><img src="../../images/OSS_File_Submission.png" /></a>

## Implementation

There are two main methods for submitting files to ACHGateway: HTTP or stream. Files can also be canceled. Each file needs to have a `shardKey` and `fileID`. Refer to [our guide on sharding](../shards/) for more context.

- `shardKey`: This is a many-to-one identifier used for assigning the shard.
- `fileID`: A unique identifier for this file.

### HTTP

ACHGateway has an endpoint for submitting a file to be queued. Refer to the [endpoint docs](https://moov-io.github.io/achgateway/api/#post-/shards/-shardKey-/files/-fileID-) for more details.

```
POST /shards/{shardKey}/files/{fileID}
```

- Content-Type: `text/plain` (default)
   - Body: Nacha format
- Content-Type: `application/json`
   - Body: moov-io/ach JSON format

The request body may be a [Nacha formatted](https://github.com/moov-io/ach/blob/master/test/testdata/ppd-debit.ach) file or the [moov-io/ach JSON representation](https://github.com/moov-io/ach/blob/master/test/testdata/ppd-valid.json). The incoming file must pass Nacha validation rules enforced by the moov-io/ach library.

### Stream

ACHGateway can accept files over a "stream" implementation supported by `gocloud.dev/pubsub`. The most common implementation is Kafka and the event format is JSON described by the [`models` package provided with ACHGateway](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models).

Submit `QueueACHFile` events:

```
{
  "id": "uuid",
  "shardKey": "uuid",
  "file": {
    ...
  }
}
```

#### Notes

Kafka topics need to be created outside of ACHGateway. Consider your needs around partitions, retention, and checkpointing when creating topics.

Make sure to understand the implications of enabling/disabling consumer groups with your kafka subscription and multiple instances of ACHGateway.

## Encryption

Both submission implementations can accepted encoded and encrypted files. This is often required to meet compliance rules. Refer to the [`compliance` package provided with ACHGateway](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/compliance) for protecting files prior to submission.

**Example**: Specify the [`Transform` config section](../../config/#inbound)

## Upload Receipt

After pending files are uploaded to the remote server a `FileUploaded` event is emitted.

```
{
    "fileID": "uuid",
    "shardKey": "uuid",
    "filename": "BANK_ACH_UPLOAD_20220601_123051.ach",
    "uploadedAt": "timestamp"
}
```

# Canceling Files

### HTTP

Make an HTTP request to cancel the file. This request can be made before the file is submitted to ACHGateway. Refer to the [endpoint docs](https://moov-io.github.io/achgateway/api/#delete-/shards/-shardKey-/files/-fileID-) for more details.

```
DELETE /shards/{shardKey}/files/{fileID}
```

### Stream

Publish a [`CancelACHFile`](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models#CancelACHFile) event to cancel a submitted file. The canceling can arrive before the `QueueACHFile` event and will still cancel the file.

```
{
  "id": "uuid",
  "shardKey": "uuid"
}
```

# Additional Notes

- Refer to the [merging operations](../../ops/merging/) page for more details on pending file storage.
