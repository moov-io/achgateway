---
layout: page
title: Merging
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Pending Files and Merging

ACHGateway is designed to accept ACH files to batch and merge for upload to a remote FTP/SFTP server. This allows operators to send multiple files over time that can be consolidated into fewer files when uploaded. The design also works without a database being required.

By persisting files to disk ACHGateway is able to survive restarts without losing data or needing to reconsume events. This helps handle higher volumes of files and allows operators access into the pending data. Filenames can be arbitrary strings (e.g. UUID's) to help identify files or objects. Pending files can be marked as canceled. Merged files optimize network usage and billing costs from your ACH partner.

ACHGateway aims to upload all possible files and accumulates errors for alerting later. A single per-file error never aborts the rest of the cutoff: later merged files still upload and still emit `FileUploaded`. If files cannot be merged they'll be uploaded individually and if a merged file cannot be flattened it'll be uploaded unflattened. A failure writing the local `uploaded/` cache is recorded for operators but does **not** block the ODFI upload — money movement takes priority.

When `ach.MergeDir` fails, the isolated cutoff directory is left intact so operators can inspect and recover the original input files. A successful empty merge (no pending ACH files) removes the isolated directory instead of recreating an empty `uploaded/` folder.

## Encryption

ACHGateway supports encrypting pending and merged files in the filesystem used for staging. This uses the [moov-io/cryptfs](https://github.com/moov-io/cryptfs) library and can be configured to use AES and encoded in base64 on disk.

## Merging

<a href="../../images/OSS_Merging_Process.png"><img src="../../images/OSS_Merging_Process.png" /></a>

When a shard is triggered ACHGateway will perform a series of steps to merge and upload the pending files.

1. Rename the existing directory of pending files from `storage/mergable/{shardName}/` to a timestamp version (e.g. `storage/{shardName}-$timestamp/`).
1. Snapshot the expected input filenames (non-canceled `*.ach` files) before any local cache writes.
1. Merge pending files (inside `storage/{shardName}-$timestamp/*.ach`) that do not contain a `*.canceled` file.
   1. With moov-io/ach's `MergeDir(...)` function (and optional `ach.Conditions` for max dollar amounts in a file, etc)
1. Build an entry-identity mapping from each input file so `FileUploaded` can be emitted for every original `fileID`. Distinct `fileID`s that share identical Nacha entry content are both uploaded and both mapped (they are not silently deduplicated).
1. Optionally `FlattenBatches()` on files and encrypt file contents (e.g. GPG)
1. Best-effort save of each merged file to `uploaded/*.ach` inside of our `storage/{shardName}-$timestamp/` directory
1. Render filename from template, prepare output formatting, and upload each prepared file. Per-file upload failures are recorded and the remaining files still upload.
1. Match each expected input back to a successfully uploaded file. Inputs that cannot be mapped (and are not already explained by a prep/upload error) fail the cutoff so operators know `FileUploaded` will not be emitted for them.

ACH transfers are merged (grouped) according to their file header values using [`ach.MergeDir`](https://pkg.go.dev/github.com/moov-io/ach#MergeDir). EntryDetail records are not modified as part of the merging process. Merging is done primarily to reduce the fees charged by your ODFI or The Federal Reserve.

Canceled files are skipped by matching the full path stem (`foo.ach` + `.canceled`, or the older `foo.canceled` layout). Matching is not done by basename alone, so a cancel in one directory cannot suppress an unrelated file with the same name.

## Encryption

ACHGateway supports encrypting pending and merged files in the filesystem used for staging. This uses the [moov-io/cryptfs](https://github.com/moov-io/cryptfs) library and can be configured to use AES and encoded in base64 on disk.

### Options

Merging files accepts a few parameters to tweak uploaded files. This allows for non-standard fields and optimized files. ACHGateway does not modify EntryDetail records, so Trace Numbers can be used to identify records. Multiple files will be created if duplicate Trace Numbers are found within pending files.

The moov-io/ach library [supports merge conditions](https://pkg.go.dev/github.com/moov-io/ach?utm_source=godoc#Conditions) and an ACHGateway shard can be configured to use them as well. An ACHGateway shard can also be configured to "flatten batches" which will consolidate EntryDetail records into fewer batches when their BatchHeader records are identical.

Refer to the [`Merging` section](../../config/#upload-agents) of the `Upload` config to tweak these values.

### Persistence

The recommended way to deploy ACHGateway is where each instance has a unique volume attached. When each instance of ACHGateway has a unique volume attached then ACH operations are shared across N instances and an individual instance would have 1/N'th of the traffic. This deployment model requires that instances consume unique ACH files (e.g. share a kafka consumer group).
