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

TODO(adam): link to openapi specification
