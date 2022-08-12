## v0.16.3 (Released 2022-08-12)

IMPROVEMENTS

- fix: elevate connection errors inside handleMessage

## v0.16.2 (Released 2022-08-11)

IMPROVEMENTS

- docs: mention ID on odfi events will be populated
- feat: set fileID on incoming ODFI files
- meta: upgrade to Go 1.19

## v0.16.1 (Released 2022-08-03)

IMPROVEMENTS

- fix: don't assume pagerduty config was provided

## v0.16.0 (Released 2022-07-27)

We've refreshed the [documentation site](https://moov-io.github.io/achgateway/) for ACHGateway with this release. We hope it helps to understand and operate ACHGateway. We've received a lot of community feedback that has improved the project and docs.

ADDITIONS

- alerting: added slack as a notifier
- feat: add endpoint for canceling a file
- feat: add ping route
- feat: emit `IncomingFile` events
- feat: support filtering ODFI files by their paths

IMPROVEMENTS

- api: add operationId and summary fields for docs
- build: update moov-io/base to v0.32.0 and moov-io/ach to v1.18.2
- fix: handle CancelACHFile inside the pipeline
- incoming/web: clearly return 200 on successful file submission

BUILD

- build: update github.com/moov-io/base to v0.33.0

## v0.15.7 (Released 2022-06-16)

IMPROVEMENTS

- fix: save plaintext audit files when GPG isn't configured
- fix: stop accumulating receivers on each handled message
- incoming/odfi: fix ProcessFiles to route around directories and files

## v0.15.6 (Released 2022-06-14)

IMPROVEMENTS

- fix: enable diffie-hellman-group-exchange-sha256 ssh algorithm

## v0.15.5 (Released 2022-06-10)

IMPROVEMENTS

- docs: cleanup getting started example
- fix: share consul session refresh logic, skip on nil consul client

## v0.15.4 (Released 2022-06-08)

IMPROVEMENTS

- fix: cleanup fileReceiver shutdown
- fix: send cutoff Day events on holidays
- refactor: use cryptfs for most of GPG encryption

BUILD

- build: update github.com/moov-io/ach to v1.16.1

## v0.15.3 (Released 2022-05-18)

IMPROVEMENTS

- upload: let sync fail if the server doesn't support it

BUILD

- build: update base images

## v0.15.2 (Released 2022-05-17)

IMPROVEMENTS

- build: run Go tests on macOS and Windows
- fix: sync, chmod, and then close in SFTP file upload
- storage: always close files in tests
- storage: close underlying file after decrypting contents
- test: benchmark with AES merging encryption
- test: fix path comparison on Windows
- testing: skip external tests when -short is specified

BUILD

- build: update Docker image to Go 1.18
- build: update github.com/moov-io/base to v0.29.0

## v0.15.1 (Released 2022-05-09)

BUILD

- build: update github.com/moov-io/ach to v1.15.1

## v0.15.0 (Released 2022-05-03)

ADDITIONS

- pipeline: support passing ach merge conditions through
   - Note: This moves `FlattenBatches: {}` to under a shard's `Mergable` object. See [the configuration docs](https://github.com/moov-io/achgateway/blob/v0.15.0/docs/CONFIGURATION.md#sharding) for more information.

## v0.14.0 (Released 2022-04-01)

IMPROVEMENTS

- pipeline: return the source hostname when listing pending files
- pipeline: return the status (error) of each shard after manually triggered
- pipeline: send holiday notification about skipping processing

## v0.13.2 (Released 2022-03-25)

IMPROVEMENTS

- fix: nil check on some shutdown calls
- fix: return Environment even with errors during startup
- incoming/stream: bump min kafka version to v2.6.0

## v0.13.1 (Released 2022-03-09)

IMPROVEMENTS

- notify: retry temporary email send failures
- pipeline: alert when we fail notifyAfterUpload

## v0.13.0 (Released 2022-02-15)

ADDITIONS

- upload: add a config (`SkipDirectoryCreation bool`) for ensuring directories prior to upload

IMPROVEMENTS

- upload: include full write path in error
- upload: reduce permissions needed when creating files (request `os.O_WRONLY` instead of `os.O_RDWR`)

## v0.12.1 (Released 2022-02-01)

IMPROVEMENTS

- pipeline: wire through error alerting struct

## v0.12.0 (Released 2022-01-27)

IMPROVEMENTS

- pipeline: close files opened within merging
- pipeline: save ValidateOpts alongside each file for later merging
- pipeline: update moov-io/ach and verify ValidateOpts are persisted
- pipeline: pass through ACH ValidateOpts when merging files
- pipeline: add a test and logging for filtering manual cutoffs
- upload: record SFTP retry attempts

BUILD

- build: update moov-io/ach to v1.13.0

## v0.11.1 (Released 2022-01-18)

IMPROVEMENTS

- output: support CR+LF line endings

## v0.11.0 (Released 2021-12-27)

ADDITIONS

- pipeline: add endpoints for listing pending files prior to upload
- pipeline: add pending_files metric
- storage: wire up an encrypted middle layer
- shard mappings: add endpoints for creating, listing, and getting shard mappings

IMPROVEMENTS

- pipeline: include shard name in pending file logs
- pipeline: include shard name on outbound metrics
- pipeline: pass filesystem operations through storage abstraction layer

## v0.10.4 (Released 2021-12-08)

BUG FIXES

- notify: nil guard around upload Notifications

## v0.10.3 (Released 2021-12-03)

BUG FIXES

- upload: check that one resolved IP is whitelisted

IMPROVEMENTS

- pipeline: log affirmatively when we are the leader

BUILD

- build: profile Go cpu/mem usage and upload the reports
- build: update github.com/PagerDuty/go-pagerduty to v1.4.3
- build: update github.com/ProtonMail/go-crypto
- build: update github.com/Shopify/sarama to v1.30.0
- build: update github.com/moov-io/ach to v1.12.2
- fix: update code from new linter upgrades

## v0.10.2 (Released 2021-11-16)

IMPROVEMENTS

- pipeline: attempt retries of consul leadership
- pipeline: include shard as key in log messages

## v0.10.1 (Released 2021-11-08)

BUG FIXES

- ODFI.Reconciliation accidently was reading `PatchMatcher` instead of `PathMatcher` in the YAML config.

## v0.10.0 (Released 2021-11-08)

BREAKING CHANGES

moov-io/base introduces errors when unexpected configuration attributes are found in the files parsed on startup.

BUILD

- build: update github.com/moov-io/base to v0.12.0

## v0.9.4 (Released 2021-11-01)

IMPROVEMENTS

- notify: improve formatting of values in emails and slack

## v0.9.3 (Released 2021-10-21)

IMPROVEMENTS

- pipeline: attempt to start a new session on consul errors, always alert

## v0.9.2 (Released 2021-10-13)

IMPROVEMENTS

- add TLS support for MySQL connections
- replace deprecated x/crypto/openpgp package with ProtonMail/go-crypto/openpgp

## v0.9.1 (Released 2021-09-22)

BUG FIXES

- consul: remove agent setup, simplify leader election process
- fix: include missing sprintf formatter
- notify/slack: properly format decimal amounts

## v0.9.0 (Released 2021-09-17)

IMPROVEMENTS

- consul: upgrade to 1.10 and support TLS connections
- incoming/odfi: acquire leadership prior to ODFI processing
- pipeline: better logging for ACH file handling

## v0.8.2 (Released 2021-09-14)

IMPROVEMENTS

- incoming/odfi: skip saving zero-byte files

## v0.8.1 (Released 2021-09-14)

IMPROVEMENTS

- incoming/odfi: save the ODFI files exactly are they are downloaded

## v0.8.0 (Released 2021-09-14)

ADDITIONS

- incoming/odfi: optionally store files in audit trail config

IMPROVEMENTS

- audittrail: don't overwrite files if they exist
- docs: update config section for inbound / outbound aduittrail storage
- pipeline: save uploaded files under "outbound/" root path

BUILD

- upload: fix build constraints for Go 1.17

## v0.7.1 (Released 2021-09-04)

BUG FIXES

- reconciliation: The ReconciliationFile event updated to include debit entries

## v0.7.0 (Released 2021-09-02)

ADDITIONS

- models: add SetValidation methods for each event type

IMPROVEMENTS

- models: allow reading partial files within events

BUILD

- build: upgrade github.com/moov-io/ach to v1.12.0

## v0.6.5 (Released 2021-08-26)

BUG FIXES

- service: remove unused Notifications root config
- upload: trim filename templates

## v0.6.4 (Released 2021-08-17)

BUG FIXES

- pipeline: create dir so it can be isolated if it doesn't exist

## v0.6.3 (Released 2021-08-17)

BUG FIXES

- pipeline: keep shard files isolated when merging
- streamtest: use random inmem subscription

## v0.6.2 (Released 2021-08-13)

IMPROVEMENTS

- models: mask AESConfig's Key in JSON marshaling

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
