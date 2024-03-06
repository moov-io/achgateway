---
layout: page
title: Notifications
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Notifications

ACHGateway enhances operational transparency by generating notifications for file uploads, providing teams and ODFIs with immediate insights into ACH activities.

## Email

Emails serve as a critical communication channel, especially for ODFIs that correlate notifications with ACH file uploads to avoid manual processing errors or accidental uploads.

Subject: `BANK_ACH_UPLOAD_20220601_123051.ach uploaded by Company`

```
A file has been uploaded to sftp.bank.com - BANK_ACH_UPLOAD_20220601_123051.ach
Name: Your Company
Debits:  100.01
Credits: 100.01

Batches: 1
Total Entries: 24
```

## Slack

Slack notifications offer a real-time, asynchronous method to report on the status of file uploads, enabling teams to stay informed about successful operations and address failures promptly.

**Success Notification:**

```
SUCCESSFUL upload of BANK_ACH_UPLOAD_20220601_123051.ach to sftp.bank.com:22 with ODFI server
8 entries | Debits: 3,442.66 | Credits: 3,442.66
```

**Failure Notification:**

```
FAILED upload of BANK_ACH_UPLOAD_20220601_123051.ach to sftp.bank.com:22 with ODFI server
2 entries | Debits: 31.03 | Credits: 31.03
```

## PagerDuty

PagerDuty notifications ensure that high-priority issues are immediately brought to the attention of the responsible parties, enabling rapid response and resolution to maintain seamless ACH file processing operations.
