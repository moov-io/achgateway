---
layout: page
title: Upload Agents
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Upload Agents

ACHGateway facilitates the secure and efficient uploading of ACH files to financial institutions (FIs) using protocols such as FTP(s) ([File Transport Protocol](https://en.wikipedia.org/wiki/File_Transfer_Protocol) with TLS) or SFTP ([SSH File Transfer Protocol](https://en.wikipedia.org/wiki/SSH_File_Transfer_Protocol)). The system adheres to specific filename conventions, like `FOO_YYYYMMDD_ABA.ach`, for example, `BANKNAME_20181222_301234567.ach`. These operations are fully configurable within the ACHGateway's settings.

**Further Reading**: See how to configure the [`Upload` object](../../config/#upload-agents) for detailed upload instructions.

## IP Whitelisting for Enhanced Security

To bolster security during the upload process to the ODFI server, ACHGateway supports IP whitelisting. This feature ensures the server's hostname resolves only to pre-approved IP addresses or CIDR ranges, offering a safeguard against DNS poisoning or routing errors.

### Configuring `UploadAgent`'s `AllowedIPs`:

- Specific IP Address: `35.211.43.9`
- CIDR Range: `10.4.0.0/16`
- Multiple Values: `10.1.0.12,10.3.0.0/16`

This configuration helps enforce strict network controls, providing an additional layer of security.

## SFTP Host and Client Key Verification

For secure SFTP file uploads, ACHGateway can verify the host key of the remote server and utilize a client key for mutual authentication. This double-layered approach ensures a trusted connection between ACHGateway and the remote server before any file transfer occurs.

### Remote Server's Host Key Configuration:

**Public Key** (in SSH Authorized Key Format):

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

Note: Public and private keys can either be directly used in their original formats or encoded in base64, adhering to Go's `base64.StdEncoding` (not URL encoding). This flexibility allows for secure and adaptable key management in line with your security protocols.
