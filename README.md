<!--generated-from:11badeae7f5171e6ec312a610718a7a4ac276e18df06d4d715e771702f50aba8 DO NOT REMOVE, DO UPDATE -->
[![Moov Banner Logo](https://user-images.githubusercontent.com/20115216/104214617-885b3c80-53ec-11eb-8ce0-9fc745fb5bfc.png)](https://github.com/moov-io)

<p align="center">
  <a href="https://moov-io.github.io/achgateway/">Project Documentation</a>
  ·
  <a href="https://moov.io/blog/education/ach-gateway-guide/">Quickstart Guide</a>
  ·
  <a href="https://slack.moov.io/">Community</a>
  ·
  <a href="https://moov.io/blog/">Blog</a>
  <br>
  <br>
</p>

[![GoDoc](https://godoc.org/github.com/moov-io/achgateway?status.svg)](https://godoc.org/github.com/moov-io/achgateway)
[![Build Status](https://github.com/moov-io/achgateway/workflows/Go/badge.svg)](https://github.com/moov-io/achgateway/actions)
[![Coverage Status](https://codecov.io/gh/moov-io/achgateway/branch/master/graph/badge.svg)](https://codecov.io/gh/moov-io/achgateway)
[![Go Report Card](https://goreportcard.com/badge/github.com/moov-io/achgateway)](https://goreportcard.com/report/github.com/moov-io/achgateway)
[![Apache 2 licensed](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/moov-io/achgateway/master/LICENSE)

# moov-io/achgateway

An extensible, highly-available, distributed, and fault-tolerant ACH uploader and downloader. ACH Gateway creates events for outside services and transforms files prior to upload to fit real-world requirements of production systems.

If you believe you have identified a security vulnerability please responsibly report the issue as via email to security@moov.io. Please do not post it to a public issue tracker.

## Getting started

Read through the [project docs](https://moov-io.github.io/achgateway/) to gain an understanding of this project's purpose and how to run it.

We publish a [public Docker image `moov/achgateway`](https://hub.docker.com/r/moov/achgateway/) from Docker Hub (`linux/amd64` and `linux/arm64`) or use this repository. The process listens on `:8484` (public API) and `:9494` (admin / Prometheus metrics). The default config starts those listeners, but **accepting and uploading files requires `Sharding` and `Upload` configuration**. Without them, `POST /shards/{shardKey}/files/{fileID}` returns HTTP 400 (`no aggregator for shard key … found`) and `PUT /trigger-cutoff` returns HTTP 400 (`unknown shard …`).

The admin server (`:9494`) has no authentication. Bind it to localhost or a private network. Manual operations such as `PUT /trigger-cutoff` and `PUT /trigger-inbound` are served there on purpose.

### Run from source or a binary

Requires [Go 1.26 or newer](https://golang.org/doc/install). A minimal config (no Kafka, FTP at `localhost:2121`, shard key `foo` → shard name `testing`) is in [`examples/local/config.yml`](./examples/local/config.yml). See [docs/usage/source.md](./docs/usage/source.md) and the [SFTP walkthrough](./docs/usage/sftp.md).

```
git clone https://github.com/moov-io/achgateway.git
cd achgateway

# `make build` writes ./bin/achgateway and stamps Version from the latest git tag.
# `go run` without ldflags reports Version=dev.
make install
make build
./bin/achgateway -h
./bin/achgateway -version

APP_CONFIG=./examples/local/config.yml ./bin/achgateway
# or: APP_CONFIG=./examples/local/config.yml go run ./cmd/achgateway
```

`APP_CONFIG` is merged over `configs/config.default.yml`. An optional `APP_CONFIG_SECRETS` file overrides `APP_CONFIG` (typically for credentials).

```
ts=2026-11-16T16:10:06Z msg="Loading APP_CONFIG config file" APP_CONFIG=./examples/local/config.yml app=achgateway level=info version=v0.37.1
ts=2026-11-16T16:10:06Z msg="public listening on :8484 for HTTP" app=achgateway level=info version=v0.37.1
ts=2026-11-16T16:10:06Z msg="listening on [::]:9494" app=achgateway level=info version=v0.37.1
```

### Docker (getting-started example)

The root `docker-compose.yml` does not include an `achgateway` service (only MySQL, FTP, SFTP, and Kafka). To run the gateway with FTP and Kafka, use [`examples/getting-started/`](./examples/getting-started):

```
# From ./examples/getting-started/
$ docker compose up
```

```
achgateway-1  | ts=2026-11-16T16:10:06Z msg="public listening on :8484 for HTTP" app=achgateway level=info version=v0.37.1
achgateway-1  | ts=2026-11-16T16:10:06Z msg="listening on [::]:9494" app=achgateway level=info version=v0.37.1
```

### Submit a file

Run curl (and `achcli`) from the **repository root** so `./testdata/…` resolves.

Submit a file (Nacha ACH format):

```
$ curl -sS -XPOST "http://localhost:8484/shards/foo/files/f6" --data @./testdata/ppd-debit.ach
```

```json
{
  "id": "f6",
  "shardKey": "foo",
  "error": "",
  "nextCutoff": "2026-11-16T16:30:00-05:00",
  "acceptedAt": "2026-11-16T16:10:06.92573243Z"
}
```

```
ts=2026-11-16T16:10:06Z msg="accepted ACH file" app=achgateway fileID=f6 level=info shardKey=foo shardName=testing version=v0.37.1
```

The URL uses shard **key** `foo`. `Sharding.Mappings` maps that key to shard **name** `testing` (the `shardName=testing` log field). Cutoff triggers use shard names (`testing`), not keys (`foo`).

Submit a file (moov-io ACH JSON format):

```
$ curl -sS -XPOST "http://localhost:8484/shards/foo/files/f4" --data @./testdata/ppd-valid.json
```

### Initiate cutoff time processing (upload to your ODFI)

This endpoint is on the admin port and uses the shard **name**:

```
$ curl -sS -XPUT "http://localhost:9494/trigger-cutoff" --data '{"shardNames":["testing"]}'
```

```
ts=2026-11-16T16:10:07Z msg="starting manual cutoff window processing" app=achgateway level=info shard=testing version=v0.37.1
ts=2026-11-16T16:10:07Z msg="found *upload.FTPTransferAgent agent" app=achgateway level=info shardName=testing version=v0.37.1
```

With `examples/local/config.yml` and no FTP server, this returns HTTP 400 (`connect: connection refused`). Start the FTP service from `examples/getting-started/`, or follow the [SFTP walkthrough](./docs/usage/sftp.md).

View an uploaded file with [`achcli` from moov-io/ach](https://github.com/moov-io/ach#command-line). The getting-started FTP service writes under `./testdata/ftp-server/outbound/`:

```
$ achcli ./testdata/ftp-server/outbound/<uploaded-file>.ach
```

### Initiate inbound file processing

Requires `Inbound.ODFI` (enabled in the getting-started config). Also on the admin port:

```
$ curl -sS -XPUT "http://localhost:9494/trigger-inbound"
```

## Operations and production

- [Cutoffs](./docs/ops/cutoffs.md)
- [Merging](./docs/ops/merging.md)
- [File options](./docs/ops/file-options.md)
- [File cleanup](./docs/ops/cleanup.md)
- [Production checklist](./docs/production/checklist.md)

## Usage

achgateway accepts files over HTTP and Kafka to queue them up for upload at a Nacha cutoff time. This allows systems and humans to publish files and have them be optimized for upload. achgateway is inspired by [the work done in moov-io/paygate](https://github.com/moov-io/paygate) and is used in production at Moov.

## Project status

This project is used in production at multiple companies and has reached a stable status. We are looking to improve the configuration of ACHGateway and looking for feedback from real-world usage. Please reach out and share your story.

## Getting help

 channel | info
 ------- | -------
[Project Documentation](https://moov-io.github.io/achgateway/) | Our project documentation available online.
Twitter [@moov](https://twitter.com/moov)	| You can follow Moov.io's Twitter feed to get updates on our project(s). You can also tweet us questions or just share blogs or stories.
[GitHub Issue](https://github.com/moov-io/achgateway/issues) | If you are able to reproduce a problem please open a GitHub Issue under the specific project that caused the error.
[moov-io slack](https://slack.moov.io/) | Join our slack channel (`#ach`) to have an interactive discussion about the development of the project.

## Supported and tested platforms

- 64-bit Linux (Ubuntu, Debian), macOS, and Windows

## Contributing

Yes please! Please review our [Contributing guide](CONTRIBUTING.md) and [Code of Conduct](https://github.com/moov-io/ach/blob/master/CODE_OF_CONDUCT.md) to get started! Checkout our [issues for first time contributors](https://github.com/moov-io/achgateway/contribute) for something to help out with.

This project uses [Go Modules](https://github.com/golang/go/wiki/Modules) and uses Go 1.26 or higher. See [Golang's install instructions](https://golang.org/doc/install) for help setting up Go. You can download the source code and we offer [tagged and released versions](https://github.com/moov-io/achgateway/releases/latest) as well. We highly recommend you use a tagged release for production.

### Test coverage

Improving test coverage is a great candidate for new contributors and allows the project to move more quickly by reducing regression issues that might not be caught before a release is pushed out to our users. One great way to improve coverage is by adding edge cases and different inputs to functions (or [contributing and running fuzzers](https://github.com/dvyukov/go-fuzz)).

Tests can run processes (like SQLite databases), but should only do so locally.

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
