## v0.22.1 (Released 2023-05-25)

IMPROVEMENTS

- fix: build paths correctly for audittrails
- incoming/odfi: pass through logger to maintain contextual fields
- pipeline: share logger across more calls

BUILD

- build: split docker image creation out from tests

## v0.22.0 (Released 2023-05-24)

IMPROVEMENTS

- feat: allow audittrail base paths to be configurable
- fix: Correcting Error Messaging on Publish
- incoming/odfi: don't emit IncomingFile events for empty ACH files
- pipeline: remove consul and leader election

BUILD

- chore: update github.com/cloudflare/circl to v1.3.3
- chore: update github.com/jlaffaye/ftp to v0.2.0
- chore: update github.com/moov-io/ach to v1.31.3
- chore: update github.com/moov-io/base to v0.43.0
- chore: update github.com/moov-io/cryptfs to v0.4.2
- chore: update golang.org/x/crypto to v0.9.0
- chore: update golang.org/x/sync to v0.2.0

## v0.21.0 (Released 2023-05-11)

ADDITIONS

- incoming/odfi: allow configuration of ValidateOpts

IMPROVEMENTS

- fix: pass event emitter errors to alerters
- stream: try to extract consumer and producer errors from sarama
- alerting: extract more information from PagerDuty error responses
- docs: fixup getting started example

BUILD

- chore: update github.com/moov-io/ach to v1.31.2

## v0.20.0 (Released 2023-04-18)

IMPROVEMENTS

- build: update github.com/moov-io/ach to v1.31.0
- feat: start supporting more kafka producer options
- odfi: allow unordered batches

BUILD

- docs: update gems

## v0.19.0 (Released 2023-04-10)

IMPROVEMENTS

- events: allow inmem stream for emitter
- feat: add models.ReadWithOpts for events
- test: verify odfi processor handles files with mixed returns and corrections

BUILD

- chore: update github.com/moov-io/ach to v1.30.0
- chore: update github.com/moov-io/base to v0.40.1
- chore: update github.com/rickar/cal/v2 to v2.1.13

## v0.18.2 (Released 2023-03-27)

IMPROVEMENTS

- docs: mention using absolute paths for upload agents
- fix: support nested inbound directory structures
- fix: adjusting implementation to be more explicit about supported folder processing

## v0.18.1 (Released 2023-03-15)

IMPROVEMENTS

- pipeline: cleanup "found %d matching ACH files" logs
- pipeline: consistently check and reconnect on network errors
- test: pass through ackdeadline for mem pubsub

## v0.18.0 (Released 2023-03-09)

This release of achgateway uses the `.AutoCommit` configuration option to determine when messages are acknowledged.
When enabled messages are acknowledged before processing. When disabled only successful messages are acknowledged.

IMPROVEMENTS

- pipeline: initialize shard metrics on startup
- pipeline: error log merge errors
- pipeline: Let .Autocommit determine when messages are committed

BUILD

- build: require Go 1.20.2 or newer in CI
- build: remove docker push from standard Go build
- update github.com/ProtonMail/go-crypto to v0.0.0-20230217124315-7d5c6f04bbb8
- update github.com/Shopify/sarama to v1.38.1
- update github.com/hashicorp/go-retryablehttp to v0.7.2
- update github.com/moov-io/ach to v1.29.2
- update github.com/moov-io/base to v0.39.0
- update github.com/moov-io/cryptfs to v0.4.1
- update github.com/rickar/cal/v2 to v2.1.12
- update github.com/sethvargo/go-retry to v0.2.4
- update github.com/slack-go/slack to v0.12.1
- update github.com/stretchr/testify to v1.8.2
- update golang.org/x/crypto to v0.7.0
- update golang.org/x/text to v0.8.0

## v0.17.7 (Released 2023-02-03)

IMPROVEMENTS

- pipeline: require shardNames when manually triggering cutoff windows
- pipeline: attempt to reconnect stream subscriptions on network errors
- test: verify we reconnect from flakey subscriptions

BUILD

- build: upgrade golang to 1.20

## v0.17.6 (Released 2023-01-13)

Note: moov-io/ach version v1.28.0 does not preserve spaces in fields like `DFIAccountNumber`. Enable `PreserveSpaces: true` to restore this behavior.

BUILD

- fix(build): update module github.com/moov-io/ach to v1.28.0
- fix(build): update module github.com/moov-io/base to v0.38.1
- fix(build): update module golang.org/x/text to v0.6.0

## v0.17.5 (Released 2023-01-13)

IMPROVEMENTS

- feat: support gzip compression with Transforms

BUILD

- fix(build): update module github.com/PagerDuty/go-pagerduty to v1.6.0
- fix(build): update module github.com/ProtonMail/go-crypto to v0.0.0-20221026131551-cf6655e29de4
- fix(build): update module github.com/Shopify/sarama to v1.37.2
- fix(build): update module github.com/hashicorp/consul/api to v1.18.0
- fix(build): update module github.com/moov-io/ach to v1.26.1
- fix(build): update module github.com/moov-io/base to v0.37.0
- fix(build): update module github.com/prometheus/client_golang to v1.14.0
- fix(build): update module github.com/rickar/cal/v2 to v2.1.9
- fix(build): update module github.com/slack-go/slack to v0.11.4
- fix(build): update module github.com/spf13/viper to v1.14.0
- fix(build): update module gocloud.dev to v0.26.0
- fix(build): update module gocloud.dev/pubsub/kafkapubsub to v0.26.0
- fix(build): update module golang.org/x/crypto to v0.4.0
- fix(build): update module golang.org/x/text to v0.5.0

## v0.17.4 (Released 2022-11-07)

IMPROVEMENTS

- fix: improve logging around consul election
- pipeline: log when requested shard isn't found

## v0.17.3 (Released 2022-11-02)

IMPROVEMENTS

- pipeline: fix calling of uploadFilesErrors metric

## v0.17.2 (Released 2022-10-26)

IMPROVEMENTS

- models: remove Filename from FileUploaded event
- pipeline: include holiday name and host in message
- pipeline: skip uploading files after caching fails
- shards: simplify config file mapping syntax

BUILD

- build: fix quotes in release script
- build: update moov-io base, ach and /x/text
- docs: include mappings and default shard
- meta: cleanup codeowners, require go 1.19.2, only push on moov-io
- test: Regenerate Consul Certs

## v0.16.10 (Released 2022-10-03)

BUILD

- build: upgrade github.com/hashicorp/consul/api to v1.15.2

## v0.16.9 (Released 2022-10-03)

IMPROVEMENTS

- docs: help clarify leadership
- docs: without leadership mention receiving unique files
- pipeline: log and ack unhandled messages instead of getting stuck
- pipeline: log kafka message details during failures

BUILD

- build: upgrade github.com/rickar/cal/v2 to v2.1.7

## v0.16.8 (Released 2022-09-14)

The release process of v0.16.7 failed due to some dependencies being out date.

IMPROVEMENTS

- build: require go 1.19.1
- build: upgrade github.com/PagerDuty/go-pagerduty to v1.5.1
- build: upgrade github.com/ProtonMail/go-crypto to v0.0.0-20220824120805-4b6e5c587895
- build: upgrade github.com/Shopify/sarama to v1.36.0
- build: upgrade github.com/hashicorp/consul/api to v1.14.0
- build: upgrade github.com/hashicorp/go-retryablehttp to v0.7.1
- build: upgrade github.com/jlaffaye/ftp to v0.1.0
- build: upgrade github.com/moov-io/ach to v1.19.3
- build: upgrade github.com/moov-io/base to v0.35.0
- build: upgrade github.com/ory/dockertest/v3 to v3.9.1
- build: upgrade github.com/pkg/sftp to v1.13.5
- build: upgrade github.com/sethvargo/go-retry to v0.2.3
- build: upgrade github.com/slack-go/slack to v0.11.3
- build: upgrade github.com/spf13/viper to v1.13.0
- build: upgrade golang.org/x/crypto to v0.0.0-20220829220503-c86fa9a7ed90

## v0.16.7 (Released 2022-09-14)

IMPROVEMENTS

- feat: include shard name with more error messages
- fix: bubble up more errors from file processing
- pipeline: fix interpolation of holiday message

## v0.16.6 (Released 2022-08-30)

IMPROVEMENTS

- feat: include shard on more ODFI logging
- fix: cleanup stack trace within PD alerts

## v0.16.5 (Released 2022-08-24)

IMPROVEMENTS

- fix: use proper loop variables when hashing entries
- incoming/odfi: trim spaces and newlines from files

## v0.16.4 (Released 2022-08-22)

IMPROVEMENTS

- build: update moov-io/base to v0.34.0 and moov-io/ach to v1.19.0
- feat: populate EntryDetail ID's with hash of contents
- fix: make incoming/odfi processor even more tolerant

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
