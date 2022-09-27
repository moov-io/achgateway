---
layout: page
title: File Options
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# File Options

When submitting files to ACHGateway there may be requirements which break Nacha's specification for field formats and values. These may be static values your ODFI requires or custom codes/rules that ehnance your product. The moov-io/ach library supports [custom validation options](https://moov-io.github.io/ach/custom-validation/) and the full [set of options is supported by ACHGateway](https://pkg.go.dev/github.com/moov-io/ach?utm_source=godoc#ValidateOpts).

### Events

Using the Go package in [achgateway's `models` package](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models) each event has a method to set `ValidateOpts` on the file. This will be carried through the File for use during merge and upload. Submissions should contain the same `ValidateOpts` to ensure the merged files have the correct overrides.

`func (Event) SetValidation(opts *ach.ValidateOpts)`

#### Errors

If you encounter the following errors you should verify that events sent to ACHGateway are wrapped properly. Try verifying the output of wrapping `pkg/models.Event`'s `MarshalJSON` around your specific events.

```
nil pubsub message
```
```
unhandled message
```

Example:
```
{
    "type": "QueueACHFile",
    "event": {
        "fileID": "uuid",
        "shardKey": "uuid",
        "file": {
            ...
        }
    }
}
```
