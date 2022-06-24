---
layout: page
title: Shards / Upload Agents
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Sharding

Shards are defined in ACHGateway as a logical grouping for ACH file delivery. There are countless patterns for describing file delivery within a business according to risk, feature delivery, fund availability, etc. Below is a sample sharding setup:

- `testing`: A shard used for automated and manual verification of the platform/feature. These files are never uploaded to the ODFI.
- `SD-live-bank1`: A shard for uploading ACH files according to the [Same-Day ACH windows](https://www.nacha.org/system/files/2021-03/SDA_Schedules_and_Funds_Availability.pdf).
- `ND-live-bank1`: A shard for uploading ACH files for the last Traditional ACH window.
- `SD-live-bank2`: Another FI used as an ODFI.

TODO(adam): Diagram

A business might have several customers that map to each shard. For example every new signup might be mapped to the `testing` shard until they're verified and onboarded successfully. Premium customers might be assigned to the `SD-live-bank1` window and free users may be mapped to `ND-live-bank1`.

The mapping is a `shardKey` (CustomerID, UUID, etc) to the `shardName` (e.g. `SD-live-bank1`). This configuration can be managed within ACHGateway via [HTTP endpoints](#) (TODO(adam)) or in the config file. Many implementations will also use a 1:1 mapping (`SD-live-bank1` -> `SD-live-bank1` and another database manages the mapping.

TODO(adam): Link to config

## Shard Configuration

Each shard has a wealth of configuration options, but the major options are:

- Upload Agent: SFTP or FTP transmission
- Cuttoff times: Wall clock times to trigger merge, upload, and notification
- Filename templates: Custom templating of uploaded filenames
- Audit Trail: Encrypted storage of uploaded and downloaded files
- Notifications: Email, PagerDuty, and Slack alerting of successful or failed processing
- Merge conditions: Maximum file dollar amounts, or line length restrictions
