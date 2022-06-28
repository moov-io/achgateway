---
layout: page
title: Overview
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Configuration

Custom configuration for this application may be specified via an environment variable `APP_CONFIG` to a configuration file that will be merged with the default configuration file.

- [Default Configuration](https://github.com/moov-io/achgateway/tree/master/configs/config.default.yml)
- [Config Source Code](https://github.com/moov-io/achgateway/blob/master/internal/service/model_config.go)

### General Configuration

```yaml
ACHGateway:
  Admin:
    BindAddress: <string>
```

### Database
```yaml
  Database:
    MySQL:
      Address: <string>
      User: <string>
      Password: <string>
      Connections:
        MaxOpen: <integer>
        MaxIdle: <integer>
        MaxLifetime: <duration>
        MaxIdleTime: <duration>
      UseTLS: <boolean>
      TLSCAFile: <string>
      InsecureSkipVerify: <boolean>
      VerifyCAFile: <boolean>
    SQLite:
      Path: <string>
    DatabaseName: <string>
```

### Consul

```yaml
  Consul:
    Address: <string>
    Scheme: <string>
    SessionPath: <string>

    Tags:
      - <string>

    Token: <string>
    TokenFile: <string>

    Datacenter: <string>
    Namespace: <string>

    Agent:
      ServiceCheckAddress: <string>
      ServiceCheckInterval: <duration>

    Session:
      CheckInterval: <duration>

    TLS:
      CAFile: <string>
      CertFile: <string>
      KeyFile: <string>
```

### Inbound
```yaml
  Inbound:
    HTTP:
      BindAddress: <string>
    InMem:
      URL: <string>
    Kafka:
      Brokers:
        - <string>
      Key: <string>
      Secret: <string>
      Group: <string>
      Topic: <string>
      TLS: <boolean>
      AutoCommit: <boolean>
    ODFI:
      Audit:
        ID: <string>
        BucketURI: <string>
        GPG:
          KeyFile: <string>
          Signer:
            KeyFile: <string>
            KeyPassword: <string>
      Processors:
        Corrections:
          Enabled: <boolean>
        Reconciliation:
          Enabled: <boolean>
          PathMatcher: <string>
        Prenotes:
          Enabled: <boolean>
        Returns:
          Enabled: <boolean>
      Publishing:
        Kafka:
          Brokers:
            - <string>
          Key: <string>
          Secret: <string>
          Group: <string>
          Topic: <string>
          TLS: <boolean>
          AutoCommit: <boolean>
      Interval: <duration>
      ShardNames:
        - <string>
      Storage:
        Directory: <string>
        CleanupLocalDirectory: <boolean>
        KeepRemoteFiles: <boolean>
        RemoveZeroByteFiles: <boolean>
```

### Eventing
```yaml
  Events:
    Stream:
      Kafka
        Brokers:
          - <string>
        Key: <string>
        Secret: <string>
        Group: <string>
        Topic: <string>
        TLS: <boolean>
        AutoCommit: <boolean>
    Webhook:
      Endpoint: <string>
```

### Sharding
```yaml
  Sharding:
    Shards:
      - Name: <string>
        Cutoffs:
          Timezone: <string>
          Windows:
            - <string>
        PreUpload:
          GPG:
            KeyFile: <string>
            Signer:
              KeyFile: <string>
              KeyPassword: <string>
        UploadAgent: <string>
        Mergable:
          # If Conditions is nil files are merged until reaching Nacha's limit of 10,000 lines
          Conditions:
            MaxLines: <integer>
            MaxDollarAmount: <integer>
          FlattenBatches: {}
        OutboundFilenameTemplate: <string>
        Audit:
          ID: <string>
          BucketURI: <string>
          GPG:
            KeyFile: <string>
            Signer:
              KeyFile: <string>
              KeyPassword: <string>
        Output:
          Format: <string>
        Notifications:
          Email:
            - ID: <string>
              From: <string>
              To:
                - <string>
              ConnectionURI: <string>
              Template: <string>
              CompanyName: <string>
          PagerDuty:
            - ID: <string>
              ApiKey: <string>
              From: <string>
              ServiceKey: <string>
          Slack:
            - ID: <string>
              WebhookURL: <string>
          Retry:
            Interval: <duration>
            MaxRetries: <integer>
```

### Upload Agents
```yaml
  Upload:
    Agents:
    - ID: <string>
      # Configuration for using a remote File Transfer Protocol server
      # for ACH file uploads.
      FTP:
        Hostname: <host>
        Username: <string>
        [ Password: <secret> ]
        [ CAFile: <filename> ]
        [ DialTimeout: <duration> | default = 10s ]
        # Offer EPSV to be used if the FTP server supports it.
        [ DisabledEPSV: <boolean> | default = false ]
      # Configuration for using a remote SSH File Transfer Protocol server
      # for ACH file uploads
      SFTP:
        Hostname: <host>
        Username: <string>
        [ Password: <secret> ]
        [ ClientPrivateKey: <filename> ]
        [ HostPublicKey: <filename> ]
        [ DialTimeout: <duration> | default = 10s ]
        [ MaxConnectionsPerFile: <number> | default = 1 ]
        # Sets the maximum size of the payload, measured in bytes.
        # Try lowering this on "failed to send packet header: EOF" errors.
        [ MaxPacketSize: <number> | default = 20480 ]
        [ SkipDirectoryCreation: <boolean> | default = false ]
      Paths:
        # These paths point to directories on the remote FTP/SFTP server.
        Inbound: <filename>
        Outbound: <filename>
        Reconciliation: <filename>
        Return: <filename>
      Notifications:
        Email:
          - <string>
        PagerDuty:
          - <string>
        Slack:
          - <string>
      AllowedIPs: <string>
    Merging:
      Directory: <string>
    Retry:
      Interval: <duration>
      MaxRetries: <integer>
    DefaultAgentID: <string>
```

### Notifications
```yaml
  Notifications:
    # TODO(adam)
```

### Error Alerting
```yaml
  Errors:
    PagerDuty:
      ApiKey: <string>
      RoutingKey: <string>
    Slack:
      AccessToken: <string>
      ChannelID: <string>
    Mock:
      Enabled: <boolean>
```

---
**[Next - Running](RUNNING.md)**