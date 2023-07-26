---
layout: page
title: Events
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Events

As ACHGateway uploads and retrieves files with the remote servers it will emit events. These are defined in the [`models` package](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models) and include both Submission and ODFI events.

Events may be delivered over a HTTP webhook or supported Stream provider (e.g. Kafka). Events are encoded in their JSON format and may be optionally encrypted. To reveal events the [`compliance` package can be used](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/compliance).

**See Also**: Configure the [`Events` object](../../config/#eventing)

## Examples

[`FileUploaded`](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models#FileUploaded):

```
{
  "fileID": "2d05191f-381b-4e93-b8b4-b999f892a95a",
  "shardKey": "SD-bank1-live",
  "uploadedAt": "2009-11-10T23:00:00Z"
}
```

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
