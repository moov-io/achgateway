---
layout: page
title: Audit Trail
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

## Audit Trail

Complying with Nacha regulations and ODFI requirements, ACHGateway ensures the retention of submitted files for a mandated duration. This feature is not just about regulatory compliance; it's a valuable tool for debugging and recreating specific files and entries, enhancing the reliability and traceability of transactions. ACHGateway achieves this through encryption and storage of these files in an S3-compatible storage layer. Additionally, Moov introduces the [ach-web-viewer project](https://github.com/moov-io/ach-web-viewer), a utility for browsing and displaying individual files, further simplifying audit and review processes.

### Pending Files


ACHGateway facilitates the easy management of pending files through specific endpoints. These allow for the listing and retrieval of pending file contents, streamlining the process of handling ACH transactions before final submission.

```
GET /shards/{shardName}/files
GET /shards/{shardName}/files/{filepath}
```

For more details on working with pending files, please visit the [pending file endpoints documentation](https://moov-io.github.io/achgateway/api/#tag--Operations).


### Organized Storage Layout

To ensure efficient file management and retrieval, ACHGateway adopts a systematic approach to storing files within an S3-compatible bucket. The layout is designed to differentiate easily between files received from the ODFI and those prepared for upload, as detailed below.

#### Received from the ODFI

Files retrieved from the ODFI are stored as follows:

```
/odfi/$hostname/$yyyy-mm-dd/$filename
```

Example: `/odfi/sftp.bank.com/inbound/2022-01-17/BANK_ACH_DOWNLOAD_20220601_123051.ach`


#### Uploaded to the ODFI

Files intended for upload to the ODFI follow this layout:

```
/outbound/$hostname/$dir/$yyyy-mm-dd/$filename
```

Example: `/outbound/sftp.bank.com/2022-01-17/BANK_ACH_UPLOAD_20220601_123051.ach`

This structured approach not only complies with regulatory requirements but also aids in the efficient management and retrieval of ACH files, supporting a more streamlined and secure transaction process.
