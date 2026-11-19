---
layout: page
title: Usage | Run from source or binary
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Run from source or binary

ACH Gateway can run without Docker. The default config (`configs/config.default.yml`, embedded in the binary) only starts the public HTTP server on `:8484` and the admin server on `:9494`. **Submitting and uploading files requires `Sharding` and `Upload` configuration.** Without those, `POST /shards/{shardKey}/files/{fileID}` returns HTTP 400 (`no aggregator for shard key … found`) and `PUT /trigger-cutoff` returns HTTP 400 (`unknown shard …`).

A minimal local config (no Kafka, FTP at `localhost:2121`) is in [`examples/local/config.yml`](https://github.com/moov-io/achgateway/blob/master/examples/local/config.yml). It maps shard key `foo` to shard name `testing`. An SFTP variant is in [`examples/sftp/`](https://github.com/moov-io/achgateway/tree/master/examples/sftp).

`APP_CONFIG` points at a YAML file that is merged over the default config. `APP_CONFIG_SECRETS` is an optional second file, typically used for credentials, that overrides `APP_CONFIG`. See [Configuration](/config/).

## Build and run

Requires [Go 1.26 or newer](https://golang.org/doc/install).

```sh
git clone https://github.com/moov-io/achgateway.git
cd achgateway

# Vendor modules, then build. `make build` writes ./bin/achgateway and
# stamps github.com/moov-io/achgateway.Version from the latest git tag
# (via ldflags). `go run` without ldflags reports Version=dev.
make install
make build
./bin/achgateway -version
# v0.37.1

# Or run from source (Version is "dev" unless you pass ldflags)
go run ./cmd/achgateway -version
# dev

APP_CONFIG=./examples/local/config.yml ./bin/achgateway
# or: APP_CONFIG=./examples/local/config.yml go run ./cmd/achgateway
```

`./bin/achgateway -h` prints usage and exits. `./bin/achgateway -version` prints the version and exits.

```
ts=2026-11-16T16:10:06Z msg="Loading APP_CONFIG config file" APP_CONFIG=./examples/local/config.yml app=achgateway level=info version=v0.37.1
ts=2026-11-16T16:10:06Z msg="public listening on :8484 for HTTP" app=achgateway level=info version=v0.37.1
ts=2026-11-16T16:10:06Z msg="listening on [::]:9494" app=achgateway level=info version=v0.37.1
```

The admin server (`:9494`) has no authentication. Bind it to localhost or a private network. Use it for `PUT /trigger-cutoff` and `PUT /trigger-inbound`. The public API is `:8484`.

## Submit a file

Run the following from the **repository root** (so `./testdata/…` resolves):

```sh
curl -sS -XPOST "http://localhost:8484/shards/foo/files/f6" --data @./testdata/ppd-debit.ach
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

The path uses shard **key** `foo`. `Sharding.Mappings` sends that key to shard **name** `testing` (see the `shardName=testing` log field). Cutoff triggers use shard names, not keys.

JSON bodies work too: `--data @./testdata/ppd-valid.json`.

## Trigger a cutoff

This uploads merged files to the configured FTP/SFTP agent. The command is on the **admin** port.

```sh
curl -sS -XPUT "http://localhost:9494/trigger-cutoff" --data '{"shardNames":["testing"]}'
```

With the local FTP example and no server listening on `localhost:2121`, the admin API returns HTTP 400 and a connection error. Start an FTP server (for example the `ftp` service in [`examples/getting-started/docker-compose.yml`](https://github.com/moov-io/achgateway/blob/master/examples/getting-started/docker-compose.yml)) or use the [SFTP walkthrough](/usage/sftp/).
