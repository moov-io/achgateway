---
layout: page
title: Account Validation
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# micro-deposits

Validating accounts is often done with two small credits submitted to a bank account and those amounts verified by a user. The experience can be improved by originating same-day batches so the amounts settle quickly. To implement this submission of ACH files with a shard key of `micro-deposit` could be used and configured for your ODFIs Same-Day processing windows.

The following are entry detail records you could create and submit to ACHGateway:

| Entry Type | Bank Account | Amount (in cents) |
|---|---|---|
| Debit | Your Company Checking | 7 |
| Credit | Customer Checking | 3 |
| Credit | Customer Checking | 4 |

Putting those EntryDetail records into a CCD or WEB batch:

| Batch Number | SEC Code | Service Class Code | Company Name | Identification | EntryDescription | EffectiveEntryDate |
|---|---|---|---|---|---|---|
| 1 | WEB | 200 (Mixed Debits and Credits) | Your Startup | CORPTESTER | Acct Verify | 220627 |

See also: [Example WEB file creation](https://github.com/moov-io/ach/blob/master/examples/example_webWrite_credit_test.go)
