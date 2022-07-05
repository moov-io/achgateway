---
layout: page
title: Audit Trail
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

## Audit Trail

A requirement of Nacha regulations and many ODFIs is to retain submitted files for a period of time. This is also a benefit to implementations because it allows for debugging and reproduction of files and entries. ACHGateway can encrypt and persist these files into an S3-compatiable storage layer. Moov also publishes an [ach-web-viewer project](https://github.com/moov-io/ach-web-viewer) to list and display individual files.

### Pending Files

ACHGateway offers endpoints for listing and retrieving the contents of a pending file.

```
GET /shards/{shardName}/files
GET /shards/{shardName}/files/{filepath}
```

Refer to the [pending file endpoints](https://moov-io.github.io/achgateway/api/#tag--Operations) for viewing pending files.

### Storage Layout

Inside of an S3-compatiable bucket files will be stored according to the following layout:

Files retrieved from the ODFI
```
/odfi/$hostname/$yyyy-mm-dd/$filename
```
Example: `/odfi/sftp.bank.com/inbound/2022-01-17/BANK_ACH_DOWNLOAD_20220601_123051.ach`

Files uploaded to the ODFI
```
/outbound/$hostname/$dir/$yyyy-mm-dd/$filename
```
Example: `/outbound/sftp.bank.com/2022-01-17/BANK_ACH_UPLOAD_20220601_123051.ach`
