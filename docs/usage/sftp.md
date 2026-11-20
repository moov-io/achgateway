---
layout: page
title: Usage | SFTP
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# SFTP upload walkthrough

ACH Gateway can upload merged ACH files to an SFTP server using an [Upload agent](/config/). A ready-to-run example lives in [`examples/sftp/`](https://github.com/moov-io/achgateway/tree/master/examples/sftp).

## Configuration

The SFTP agent matches the [Upload Agents](/config/) schema (`Hostname`, `Username`, `Password` or key, and remote `Paths`). The example maps shard key `foo` to shard name `testing` and uploads to `localhost:2222`:

```yaml
  Sharding:
    Shards:
      - name: "testing"
        cutoffs:
          timezone: "America/New_York"
          windows:
            - "10:30"
            - "16:30"
        uploadAgent: "local-sftp"
        output:
          format: "nacha"
    Mappings:
      - shardKey: "foo"
        shardName: "testing"
    Default: "testing"
  Upload:
    agents:
      - id: "local-sftp"
        sftp:
          hostname: "localhost:2222"
          username: "demo"
          password: "password"
        paths:
          inbound: "/inbound/"
          outbound: "/outbound/"
          reconciliation: "/reconciliation/"
          return: "/returned/"
    merging:
      directory: "./storage/"
```

`atmoz/sftp` chroots users into their home directory, so `/outbound/` is the `outbound/` folder in that home. On a server that is **not** chrooted, set `Paths` to directories that exist on that server's filesystem.

## Docker Compose (atmoz/sftp)

[`examples/sftp/docker-compose.yml`](https://github.com/moov-io/achgateway/blob/master/examples/sftp/docker-compose.yml) runs ACH Gateway and [atmoz/sftp](https://hub.docker.com/r/atmoz/sftp). The SFTP container mounts [`testdata/sftp-server/`](https://github.com/moov-io/achgateway/tree/master/testdata/sftp-server) (with `inbound/`, `outbound/`, `reconciliation/`, and `returned/`) at `/home/demo`.

From `examples/sftp/`:

```sh
docker compose up
```

From the **repository root**:

```sh
curl -sS -XPOST "http://localhost:8484/shards/foo/files/f6" --data @./testdata/ppd-debit.ach
curl -sS -XPUT "http://localhost:9494/trigger-cutoff" --data '{"shardNames":["testing"]}'
```

Uploaded files appear under `testdata/sftp-server/outbound/`. Inspect one with [`achcli`](https://github.com/moov-io/ach#command-line):

```sh
achcli ./testdata/sftp-server/outbound/<uploaded-file>.ach
```

The root `docker-compose.yml` also has an `sftp` service (`atmoz/sftp` on port 2222) used for tests; it mounts `testdata/download-test` rather than `testdata/sftp-server`. Prefer `examples/sftp/` for this walkthrough.

## Without Docker

You can point `examples/sftp/config.yml` at any SFTP server on `localhost:2222` with user `demo` / password `password` and the four path directories. A chrooted server can use the `/outbound/`-style paths above; otherwise use absolute directories on that server.
