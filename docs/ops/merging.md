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

## Merging

1. Rename the existing directory of pending files from `storage/merging/{shardKey}/` to a timestamp version (e.g. `storage/merging/{shardKey}-$timestamp/`).
1. Merge pending files (inside `storage/merging/{shardKey}-$timestamp/*.ach`) that do not contain a `*.canceled` file.
   1. With moov-io/ach's `MergeFiles(...)` function (and optional `ach.Conditions` for max dollar amounts in a file, etc)
1. Optionally `FlattenBatches()` on files and encrypt file contents (e.g. GPG)
1. Render filename from template, prepare output formatting
1. Save file to `uploaded/*.ach` inside of our `storage/merging/{shardKey}-$timestamp/` directory

ACH transfers are merged (grouped) according to their file header values using [`ach.MergeFiles`](https://godoc.org/github.com/moov-io/ach#MergeFiles). EntryDetail records are not modified as part of the merging process. Merging is done primarily to reduce the fees charged by your ODFI or The Federal Reserve.
