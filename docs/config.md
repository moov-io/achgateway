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

### Endpoint

ACHGateway has a [`GET :9494/config` endpoint](https://moov-io.github.io/achgateway/api/#get-/config) to return the full config object.

### General Configuration

```yaml
ACHGateway:
  Admin:
    BindAddress: <string> # Example :9494
```

### Database
```yaml
  Database:
    MySQL:
      Address: <string> # Example tcp(localhost:3306)
      User: <string>
      Password: <string>
      Connections: # Optional Object
        MaxOpen: <integer>
        MaxIdle: <integer>
        MaxLifetime: <duration>
        MaxIdleTime: <duration>
      [ UseTLS: <boolean> | default = false ]
      [ TLSCAFile: <string> ]
      [ InsecureSkipVerify: <boolean> | default = false ]
      [ VerifyCAFile: <boolean> | default = false]
      TLSClientCerts:
        - CertFilePath: <string>
          KeyFilePath: <string>
    SQLite:
      Path: <string>
    DatabaseName: <string>
```

### Inbound
```yaml
  Inbound:
    HTTP:
      BindAddress: <string> # Example :8484
      Transform:
        Encoding:
          [ Base64: <boolean> | default = false ]
          [ Compress: <boolean> | default = false ]
        Encryption:
          AES:
            [ Key: <string> | default = "" ]
    InMem:
      [ URL: <string> ]
    Kafka:
      Brokers:
        - <string>
      Key: <string>
      Secret: <string>
      [ Group: <string> | default = "" ]
      Topic: <string>
      TLS: <boolean>
      AutoCommit: <boolean>
      [ SASLMechanism: <string> | default = "PLAIN" ]
      Transform:
        Encoding:
          [ Base64: <boolean> | default = false ]
          [ Compress: <boolean> | default = false ]
        Encryption:
          AES:
            [ Key: <string> | default = "" ]
      Consumer: {}
      Producer:
        [ MaxMessageBytes: <number> | default = 1000000 ]
    ODFI:
      Audit:
        ID: <string>
        # BucketURI is the S3-compatiable bucket location. AWS S3 and Google Cloud Storage are supported.
        # See https://gocloud.dev/howto/blob/ for more information on configuring each cloud provide.r.
        # Example: s3://my-bucket?region=us-west-1 OR gcs://my-bucket/
        BucketURI: <string>
        BasePath: <string> # Example: "incoming"
        GPG: # Optional, but recommended
          KeyFile: <string>
          Signer:
            KeyFile: <string>
            KeyPassword: <string>
      Processors:
        Corrections:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "CORRECTION_"
          [ PathMatcher: <string> | default = "" ]
        Incoming:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "CORRECTION_"
          [ PathMatcher: <string> | default = "" ]
          # Booleans to skip emitting events for files containing the specified record type
          [ ExcludeCorrections: <boolean> | default = false ]
          [ ExcludePrenotes: <boolean> | default = false ]
          [ ExcludeReturns: <boolean> | default = false ]
          [ ExcludeReconciliations: <boolean> | default = false ]
        Reconciliation:
          [ Enabled: <boolean> | default = false ]
          # Partial filename to match on. Example: "RECON_"
          [ PathMatcher: <string> | default = "" ]
          [ ProduceFileEvents: <boolean> | default = false ]
          [ ProduceEntryEvents: <boolean> | default = false ]
        Prenotes:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "PRENOTE_"
          [ PathMatcher: <string> | default = "" ]
        Returns:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "RET_"
          [ PathMatcher: <string> | default = "" ]
        Validation:
          # See moov-io/ach's ValidateOpts for the full list of options
      Publishing:
        Kafka:
          Brokers:
            - <string>
          Key: <string>
          Secret: <string>
          Topic: <string>
          [ TLS: <boolean> | default = false ]
          [ AutoCommit: <boolean> | default = false ]
      Interval: <duration>
      ShardNames:
        - <string>
      Storage:
        Directory: <string>
        [ CleanupLocalDirectory: <boolean> | default = false]
        [ KeepRemoteFiles: <boolean> | default = false]
        [ RemoveZeroByteFiles: <boolean> | default = false]
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
        Topic: <string>
        [ TLS: <boolean> | default = false ]
        [ AutoCommit: <boolean> | default = false ]
        [ SASLMechanism: <string> | default = "PLAIN" ]
        [ AWSRegion: <string> | default = "" ]
        [ AWSProfile: <string> | default = "" ]
        [ AWSRoleARN: <string> | default = "" ]
        [ AWSSessionName: <string> | default = "" ]
    Webhook:
      [ Endpoint: <string> | default = "" ]
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
          GPG: # Optional
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
          FlattenBatches: {} # Specify a non-null object to flatten batches, often not needed
        OutboundFilenameTemplate: <string>
        Audit:
          ID: <string>
          # BucketURI is the S3-compatiable bucket location. AWS S3 and Google Cloud Storage are supported.
          # See https://gocloud.dev/howto/blob/ for more information on configuring each cloud provide.r.
          # Example: s3://my-bucket?region=us-west-1 OR gcs://my-bucket/
          BucketURI: <string>
          BasePath: <string> # Example: "outgoing"
          GPG:
            KeyFile: <string>
            Signer:
              KeyFile: <string>
              KeyPassword: <string>
        Output:
          Format: <string> # Example nacha, base64, encrypted-bytes
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
    [ Default: <string> ]
    Mappings:
      - ShardKey: <string>  # Can be random value (UUID, fixed string)
        ShardName: <string> # Maps to Sharding.Shards[_].name
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
        # It is recommended to use absolute paths (e.g. /home/user/outbound/)
        # where possible to correctly identify filesystem locations.
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
      Storage:
        Filesystem:
          [ Directory: <strong> | default = "" ]
        Encryption:
          AES:
            [ Base64Key: <string> | default = "" ]
          Encoding: <string> # Example: base64
    Retry:
      Interval: <duration>
      MaxRetries: <integer>
    DefaultAgentID: <string>
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
