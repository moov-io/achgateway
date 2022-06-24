---
layout: page
title: ODFI Files
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# ODFI Files

ACH operates with a bi-directional set of messages. There are several message types which come from your ODFI (or the Federal Reserve) defined by Nacha: Corrections (NOCs), Returns and pre-notes. Moov has identified two additional message types used in lots of ACH implementations, such as incoming files, and reconciliations. All of these messages [are defined in the `models` package](https://pkg.go.dev/github.com/moov-io/achgateway/pkg/models).

- `CorrectionFile`: Nacha defined "Notification of Change" (NOC) batches and entries. Think EntryDetails and Addenda98s.
- `IncomingFile`: ACH files coming from other FIs. Often to debit accounts you're the processor for.
- `PrenoteFile`: Nacha defined account verification method.
- `ReconciliationFile`: Partial ACH files containing Batch header/trailer blocks with EntryDetails records. Used to signify balance clearing and settlement.
- `ReturnFile`: Nacha defined Return batches and entries. Think EntryDetails and Addenda99s

## Correction File

Correction Files (NOCs) are files with "Notification of Change" entries within them. These are used to advise originators of data updates. Often RDFI's send these to notify originators about account/routing number changes, individual name updates, or other data to update. Debits and Credits still post to their respective accounts. For more details refer to the [moov-io/ach page for Corrections](https://moov-io.github.io/ach/changes/).

## Incoming File

Many implementations will receive ACH files from other originators that impact the bank accounts the implementation controls. These are often specific to your use-case, risk, and business.

## Prenote File

Nacha has defined a "prenote" as a zero-dollar EntryDetail used to verify an account exists and is authorized to be transacted with. Not every vendor or FI supports prenotes.

## Reconciliation File

Reconciliation files is a term defined with ACHGateway to signify a partial ACH file used to signify balance clearing and settlement. Often ODFIs can deliver credit/debit entries which correspond to balance activity on accounts at the ODFI. Not every vendor or FI supports reconciliation files.

## Return File

Returns are Nacha defined Entry Detail records that have failed to post against the given account/routing number. There are lots of return codes used to indicate specific reasons. For more details refer to the [moov-io/ach page on Returns](https://moov-io.github.io/ach/returns/).
