---
layout: page
title: Production Advice
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Production Advice

Below is a list of suggestions when deploying ACHGateway in production. These tips will help secure the data used in files and submitted to the ODFI(s). There are also performance and reliability suggestions listed below.

## HTTP Servers

ACHGateway is not designed to be exposed to the public internet or a broad access point. The admin endpoints can trigger real money movement and you should understand the impact of files uploaded.

## Encryption Settings

Specify the `Transform.Encryption` config on [Inbound files](../../config/#inbound) and Merging objects.

## Upload Agents

Use TLS when connecting to upload agents. Use strong passwords and/or keys with remote FTP/SFTP servers. Verfiy DNS records resolve to expected IPs.

## Audit Trail

Encrypt files uploaded to the audit trail and keep the bucket/files private.

## MySQL Database

Use [TLS when connecting to ](../../config/#database) MySQL and use strong passwords and/or certificate keys.

## Merging

Backup the storage directory for merged files or have it recoverable in real-time.
