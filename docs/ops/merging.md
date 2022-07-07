---
layout: page
title: Merging
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Pending Files and Merging

When a shard is triggered (either by [cutoff time](../cutoffs/) or manual) the instance of ACHGateway that [is the leader](../leadership/) performs merging and uploading of the pending files to a remote server. This optimizes network usage and billing costs from your ACH partner.

## Encryption

ACHGateway supports encrypting pending and merged files in the filesystem used for staging. This uses the [moov-io/cryptfs](https://github.com/moov-io/cryptfs) library and can be configured to use AES and encoded in base64 on disk.

## Merging

1. Rename the existing directory of pending files from `storage/merging/{shardKey}/` to a timestamp version (e.g. `storage/merging/{shardKey}-$timestamp/`).
1. Merge pending files (inside `storage/merging/{shardKey}-$timestamp/*.ach`) that do not contain a `*.canceled` file.
   1. With moov-io/ach's `MergeFiles(...)` function (and optional `ach.Conditions` for max dollar amounts in a file, etc)
1. Optionally `FlattenBatches()` on files and encrypt file contents (e.g. GPG)
1. Render filename from template, prepare output formatting
1. Save file to `uploaded/*.ach` inside of our `storage/merging/{shardKey}-$timestamp/` directory

ACH transfers are merged (grouped) according to their file header values using [`ach.MergeFiles`](https://godoc.org/github.com/moov-io/ach#MergeFiles). EntryDetail records are not modified as part of the merging process. Merging is done primarily to reduce the fees charged by your ODFI or The Federal Reserve.

### Options

Merging files accepts a few parameters to tweak uploaded files. This allows for non-standard fields and optimized files. ACHGateway does not modify EntryDetail records, so Trace Numbers can be used to identify records. Multiple files will be created if duplicate Trace Numbers are found within pending files.

The moov-io/ach library [supports merge conditions](https://pkg.go.dev/github.com/moov-io/ach?utm_source=godoc#Conditions) and an ACHGateway shard can be configured to use them as well. An ACHGateway shard can also be configured to "flatten batches" which will consolidate EntryDetail records into fewer batches when their BatchHeader records are identical.

Refer to the [`Merging` section](../../config/#upload-agents) of the `Upload` config to tweak these values.

### Persistence

There are two methods for deploying ACHGateway with a persistent storage attached. Each instance of ACHGateway having a unique volume attached or the instances share one volume. Both methods have advantages and drawbacks.

**Unique Volumes**

When each instance of ACHGateway has a unique volume attached the fault tolerance of ACH operations can be higher when other instances pickup the slack. With this deployment operators will need to decide between having ACHGateway instances consume all submitted files or a subset. If multiple instances consume all submitted files then [leader election](../leadership/) is recommended in order to avoid duplicated uploads. If instances a subset of files (e.g. by specifying a Kafka consumer group) then leader election is not recommended as each instance needs to upload its share of pending files.

**Shared Volume**

A shared volume between multiple ACHGateway instances offers a benefit where you can have several consumers handling the submitted files and optional [leader election](../leadership/) during uploads. Operators should be aware of duplicate uploads when instances do not perform leader election but share the underlying volume. Shared volumes will need to handle the combined I/O operations of all instances. Not all cloud providers support this "many write, many read" deployment for volumes.

> Note: Moov has not tested running ACHGateway with a shared volume.
