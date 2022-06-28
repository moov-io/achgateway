---
layout: page
title: Project Goals
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Project Goals

achgateway is an automated service for uploading and downloading Nacha formatted ACH files to FTP/SFTP servers. This service accepts valid Nacha files across multiple interfaces and will optimize them to upload to an ODFI.

Several other features of achgateway include:

- Extensible submission of ACH files (and partial requests) for upload at cutoff times
- Merging pending files together for optimized network usage and pricing by required times during the day
- Custom filename templating on uploaded files and non-compliant Nacha validation
- Audit storage of uploaded and downloaded files and [retrieval or viewing](https://github.com/moov-io/ach-web-viewer)
- Notifications on successful file upload or errors
   - Slack, PagerDuty, Emails, etc

# Non-Goals

- Nacha compliant limit analysis
- Balance verification
- Transaction authorization
- Settlement availability
- Risk calculations
   - For OFAC, sanction, and watchlist scanning see [Watchman](https://github.com/moov-io/watchman)
