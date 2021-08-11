## v0.6.1 (Released 2021-08-11)

BUG FIXES

- events: pass in config transfer to stream service

IMPROVEMENTS

- meta: fixup from adding gosec linter

## v0.6.0 (Released 2021-08-04)

ADDITIONS

- audittrail: save agent hostname in blob path
- inbound: support TLS over http
- upload: offer ShardName and Index for filename templates

BUG FIXES

- web: fix hand-over of events through compliance protection

BUILD

- docs: mention nacha and moov-io/ach json formats

## v0.5.2 (Released 2021-08-03)

BUG FIXES

- pipeline: check incoming ACHFile is valid prior to accepting

BUILD

- build: update go.mod / go.sum
- build: use debian stable's slim image

## v0.5.1 (Released 2021-07-15)

IMPROVEMENTS

- models: expose incoming ACHFile and CancelACHFile
- service: remove outdated ODFI Publishing config

## v0.5.0 (Released 2021-07-14)

ADDITIONS

- compliance: add functions for securing reading/writing events

BUILD

- build: upgrade deps in docker images

## v0.4.3 (Released 2021-06-28)

This release contains MacOS and Windows binaries.

## v0.4.2 (Released 2021-06-18)

IMPROVEMENTS

- docs: getting started example

BUG FIXES

- pipeline: properly return nil error from emitFilesUploaded

## v0.4.1 (Released 2021-06-15)

BUG FIXES

- incoming/odfi: fix nil panic on sending events

## v0.4.0 (Released 2021-06-11)

IMPROVEMENTS

- events: move models into exported package

## v0.3.0 (Released 2021-06-11)

ADDITIONS

- events: setup events for incoming ODFI files (Corrections, Incoming, Prenotes, Reconciliation, Returns)

IMPROVEMENTS

- config: better validation error messages
- pipeline: allow for a default shard

## v0.2.2 (Released 2021-06-09)

BUG FIXES

- configs: disable mysql and consul by default

## v0.2.1 (Released 2021-06-09)

BUILD

- Remove default MySQL and Consul configurations

## v0.2.0 (Released 2021-06-08)

ADDITIONS

- pipeline: add metrics for file_receiver actions
- pipeline: emit file uploaded event if configured
- server: add an admin route for displaying the config

IMPROVEMENTS

- service: update default filename template to include seconds
- shards: read a static set of mappings from our database

BUG FIXES

- build: upgrade moov-io/ach to v1.9.1
- pipeline: guard nil xfer alerting call

## v0.1.3 (Released 2021-06-05)

BUILD

- Fix issues with releases

## v0.1.0 (Released 2021-06-04)

Initial Release

- File submission via HTTP or Kafka
- ACH merging and flattening prior upload
- Cutoff times for automated file uploads
- Shard based isolation and logical grouping
- Leader election (backed by Consul) for upload coordination
