---
layout: page
title: Notifications
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Notifications

ACHGateway can produce notifications when files are uploaded. These are helpful to confirm within a team or ODFI of intended ACH activity.

## Email

Many ODFIs expect a notification email to correlate with an uploaded ACH file. This helps to prevent manual uploads or accidents.

Example:
```
A file has been uploaded to sftp.bank.com - BANK_ACH_UPLOAD_20220601_123051.ach
Name: Your Company
Debits:  100.01
Credits: 100.01

Batches: 1
Total Entries: 24
```

## Slack

Teams can receive messages of successful or failed file uploads. These can serve as async activity reporting.

Success:
```
SUCCESSFUL upload of BANK_ACH_UPLOAD_20220601_123051.ach to sftp.bank.com:22 with ODFI server
8 entries | Debits: 3,442.66 | Credits: 3,442.66
```

Failure:
```
FAILED upload of BANK_ACH_UPLOAD_20220601_123051.ach to sftp.bank.com:22 with ODFI server
2 entries | Debits: 31.03 | Credits: 31.03
```

## PagerDuty

TODO(adam): Consolidate errors and Critical notifications
