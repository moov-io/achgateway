---
layout: page
title: Upload Agents
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Upload Agents

ACH files which are uploaded to another FI primarily use FTP(s) ([File Transport Protocol](https://en.wikipedia.org/wiki/File_Transfer_Protocol) with TLS) or SFTP ([SSH File Transfer Protocol](https://en.wikipedia.org/wiki/SSH_File_Transfer_Protocol)) and follow a filename pattern like: `FOO_YYYYMMDD_ABA.ach` (example: `BANKNAME_20181222_301234567.ach`). The configuration file determines how ACHGateway uploads and transforms the files.

**See Also**: Configure the [`Upload` object](../../config/#upload-agents)

### Filename templates

ACHGateway supports templated naming of ACH files prior to their upload. This is helpful for ODFI's which require specific naming of uploaded files.Templates use Go's [`text/template` syntax](https://golang.org/pkg/text/template/) and are validated when ACHGateway starts or changed via admin endpoints.

Example:

{% raw %}
```
{{ .ShardName }}-{{ date "20060102" }}-{{ .Index }}.ach{{ if .GPG }}.gpg{{ end }}
```
{% endraw %}

The following fields are passed to templates giving them data to build a filename from:

- `ShardName`: string of the shard performing an upload
- `GPG`: boolean if file is encrypted
- `Index`: integer starting from 0 of the Nth file uploaded during a cutoff from an ACHGateway instance

Also, several functions are available (in addition to Go's standard template functions)

- `date` Takes a Go [`Time` format](https://golang.org/pkg/time/#Time.Format) and returns the formatted string
- `env` Takes an environment variable name and returns the value from `os.Getenv`.
- `lower` and `upper` convert a string into lowercase or uppercase

### IP Whitelisting

When ACHGateway uploads an ACH file to the ODFI server it can verify the remote server's hostname resolves to a whitelisted IP or CIDR range.
This supports certain network controls to prevent DNS poisoning or misconfigured routing.

Setting an `UploadAgent`'s `AllowedIPs` property can be done with values like: `35.211.43.9` (specific IP address), `10.4.0.0/16` (CIDR range), `10.1.0.12,10.3.0.0/16` (Multiple values)

### SFTP Host and Client Key Verification

ACHGateway can verify the remote SFTP server's host key prior to uploading files and it can have a client key provided. Both methods assist in
authenticating ACHGateway and the remote server prior to any file uploads.

**Public Key** (SSH Authorized key format)

```
SFTP Config: HostPublicKey
Format: ssh-rsa AAAAB...wwW95ttP3pdwb7Z computer-hostname
```

**Private Key** (PKCS#8)

```
SFTP Config: ClientPrivateKey

Format:
-----BEGIN RSA PRIVATE KEY-----
...
33QwOLPLAkEA0NNUb+z4ebVVHyvSwF5jhfJxigim+s49KuzJ1+A2RaSApGyBZiwS
...
-----END RSA PRIVATE KEY-----
```

Note: Public and Private keys can be encoded with base64 from the following formats or kept as-is. ACHGateway expects Go's `base64.StdEncoding` encoding (not base64 URL encoding).
