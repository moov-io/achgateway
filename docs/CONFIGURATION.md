<!-- generated-from:c55598a61b9a43ce98e8500432e597669da2f8955b40d9e772eac6011e2bb1c5 DO NOT REMOVE, DO UPDATE -->
# ACH Gateway
**[Purpose](README.md)** | **Configuration** | **[Running](RUNNING.md)**

---

## Configuration
Custom configuration for this application may be specified via an environment variable `APP_CONFIG` to a configuration file that will be merged with the default configuration file.

- [Default Configuration](../configs/config.default.yml)
- [Config Source Code](../pkg/service/model_config.go)

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
      FlattenBatches: {}
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
    Mock:
      Enabled: <boolean>
```

---
**[Next - Running](RUNNING.md)**
