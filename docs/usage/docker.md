---
layout: page
title: Usage | Docker
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Docker

You can download a [docker image called `moov/achgateway`](https://hub.docker.com/r/moov/achgateway/) from Docker Hub or use this repository. Release images are published for `linux/amd64` and `linux/arm64`.

The image listens on `:8484` (public API) and `:9494` (admin / metrics) with no extra config, but **submitting and uploading files requires `Sharding` and `Upload` configuration**. The root `docker-compose.yml` only starts supporting services (MySQL, FTP, SFTP, Kafka) — it does not include an `achgateway` service.

The example that runs ACH Gateway plus FTP and Kafka is [`examples/getting-started/`](https://github.com/moov-io/achgateway/tree/master/examples/getting-started). It mounts `examples/getting-started/config.yml` via `APP_CONFIG`. To run without Docker, see [Run from source or binary](/usage/source/). For SFTP instead of FTP, see the [SFTP walkthrough](/usage/sftp/).

```
# From ./examples/getting-started/
$ docker compose up
```

```
achgateway-1  | ts=2026-11-16T16:10:06Z msg="public listening on :8484 for HTTP" app=achgateway level=info version=v0.37.1
achgateway-1  | ts=2026-11-16T16:10:06Z msg="listening on [::]:9494" app=achgateway level=info version=v0.37.1
```

The admin port (`:9494`) has no authentication. Do not expose it publicly.

### Submitting a file

Run curl from the **repository root** so `./testdata/…` resolves (those files are not inside `examples/getting-started/`).

Submit a file to achgateway (Nacha ACH format):

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
achgateway-1  | ts=2026-11-16T16:10:06Z msg="accepted ACH file" app=achgateway fileID=f6 level=info shardKey=foo shardName=testing version=v0.37.1
```

The URL uses shard **key** `foo`. `Sharding.Mappings` in the getting-started config maps that key to shard **name** `testing`. Cutoff triggers use shard names.

Submit a file to achgateway (moov-io ACH JSON format):

```
$ curl -sS -XPOST "http://localhost:8484/shards/foo/files/f4" --data @./testdata/ppd-valid.json
```

### Uploading files (to your ODFI)

Initiate cutoff time processing (aka upload to your ODFI) on the admin port, with the **shard name**:

```
$ curl -sS -XPUT "http://localhost:9494/trigger-cutoff" --data '{"shardNames":["testing"]}'
```

```
achgateway-1  | ts=2026-11-16T16:10:07Z msg="starting manual cutoff window processing" app=achgateway level=info shard=testing version=v0.37.1
achgateway-1  | ts=2026-11-16T16:10:07Z msg="found *upload.FTPTransferAgent agent" app=achgateway level=info shardName=testing version=v0.37.1
```

### View the uploaded file

With the getting-started FTP service, uploads land under `testdata/ftp-server/outbound/` (mounted into the FTP container). Use [`achcli` from moov-io/ach](https://github.com/moov-io/ach#command-line) from the repository root:

```
$ achcli ./testdata/ftp-server/outbound/<uploaded-file>.ach
```

### Process files from your ODFI

The getting-started config enables `Inbound.ODFI`. Trigger inbound processing on the admin port:

```
$ curl -sS -XPUT "http://localhost:9494/trigger-inbound"
```
