---
layout: page
title: Events
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Events

As ACHGateway uploads and retrieves files with the remote servers it will emit events. These are defined in the [`models` package](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models) and include both Submission and ODFI events.

Events can be dispatched via HTTP webhooks or through a supported streaming provider (e.g., Kafka), with all events being formatted in JSON. For added security, event data may also undergo optional encryption. The decryption and interpretation of these events are facilitated by the [`compliance` package](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/compliance).

**See Also**: Configure the [`Events` object](../../config/#eventing)

## Event Examples

### `FileUploaded` Event

This event signifies the successful upload of an ACH file to the server:

[`FileUploaded`](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models#FileUploaded):

```
{
  "fileID": "2d05191f-381b-4e93-b8b4-b999f892a95a",
  "shardKey": "SD-bank1-live",
  "filename": "SD-BANK1-LIVE-20240201-111500-1.ach",
  "uploadedAt": "2009-11-10T23:00:00Z"
}
```

### `InvalidQueueFile` Event

This event alerts to a problem with a file in the queue, such as a structural or validation error:

[`InvalidQueueFile`](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models#InvalidQueueFile):

```
{
    "file": {
        "id": "01d5af6b-0f77-4976-b681-69947ccc9ea1",
        "shardKey": "SD-bank1-live",
        "file": {
            // ach.File JSON
        }
    },
	"error": "batches out of order"
}
```
