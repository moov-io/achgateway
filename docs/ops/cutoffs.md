---
layout: page
title: Cutoffs
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Cutoff Times

A cutoff time is a wall-clock time that ACH files must be delivered to the Federal Reserve. As ACH is a batch payment method these cutoff times are the key component to batching payments. The Federal Reserve [publishes their Processing Schedule](https://www.frbservices.org/resources/resource-centers/same-day-ach/fedach-processing-schedule.html) but ODFIs typically require uploads 15-30mins prior to the Federal Reserve window. The Federal Reserve also [publishes a list of holidays](https://www.frbservices.org/about/holiday-schedules) where processing does not occur.

Example with 30-min ODFI deadline

| Schedule | ODFI Deadline | Fed Deadline | Target Distribution | Settlement Schedule |
|----|----|----|----|----|
| Same-Day | 2:15pm ET | 2:45pm ET | 4:00pm ET | 5:00pm ET |
| Future Date | 4:15pm ET | 4:45pm ET | 5:30 pm ET | 8:30 am ET (Next Day) |

## Next cutoff

When a file is queued, ACHGateway predicts the next automated cutoff that should upload it and returns that time as `nextCutoff` on the HTTP queue response. The prediction uses the shard's `Cutoffs.Timezone`, `Windows`, and `On` setting:

- `On: banking-days` (default) skips weekends and Federal Reserve holidays
- `On: all-days` includes holidays but still skips weekends, matching automated processing

A file queued at an exact window time is scheduled for a later window that day, or the first window on the next processing day. Manual [`/trigger-cutoff`](#manual-triggers) processing can still upload the file earlier than `nextCutoff`.

## Developers

Moov publishes [a `Time` object in moov-io/base](https://pkg.go.dev/github.com/moov-io/base?utm_source=godoc#Time) to assist with calculating banking days and when holidays are observed. There is also a [`bankcron` Docker image](https://github.com/moov-io/bankcron) for running tasks only on banking days.

## Manual Triggers

ACHGateway supports manually triggering inbound or cutoff processing. A list of shards can be specified or all shards can be triggered.

### Flushing ACH Files

There is an endpoint to initiate cutoff processing as if a window has approached. This involves merging transfers into files, upload attempts, and audit trail storage.

```
$ curl -XPUT http://localhost:9494/trigger-cutoff --data '{"shardNames":["testing"]}'
{
  "shards": {
    "testing": null,
    "SD-live": "ERROR: unknown host"
  }
}
```

### Processing ODFI Files

There is an endpoint to initiate processing of ODFI files which could be incoming transfers, returned files, corrected files, and pre-notifications.

```
$ curl -XPUT http://localhost:9494/trigger-inbound
{
  "shards": {
    "testing": null,
    "SD-live": "ERROR: unknown host"
  }
}
```
