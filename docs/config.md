---
layout: page
title: Overview
hide_hero: true
show_sidebar: false
menubar: docs-menu
---

# Configuration

Custom configuration for this application may be specified via an environment variable `APP_CONFIG` to a configuration file that will be merged with the default configuration file.

- [Example Configuration](https://github.com/moov-io/achgateway/tree/master/examples/getting-started/config.yml)
- [Config Source Code](https://github.com/moov-io/achgateway/blob/master/internal/service/model_config.go)

### Endpoint

ACHGateway has a [`GET :9494/config` endpoint](https://moov-io.github.io/achgateway/api/#get-/config) to return the full config object.

## Configuration

```yaml
ACHGateway:
  Admin: <AdminConfig>
  HTTP: <HTTPConfig>
  Database: <DatabaseConfig>
  Consul: <ConsumConfig>
  ODFI: <ODFI>
  RDFI: <RDFI>
  FileAgents: <FileAgents>
  Sharding: <Sharding>
  Errors: <Errors>
  Notifications: <Notifications>
```

### Admin

```yaml
  Admin:
    BindAddress: <string> # Example :9494
```

### DatabaseConfig
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
    DatabaseName: <string>
```

### Consul

```yaml
  Consul: # Optional Object
    Address: <string> # Example http://127.0.0.1:8500
    [ Scheme: <string> | default = "" ]
    SessionPath: <string>
    Tags: # Optional
      - <string>
    [ Token: <string> ]
    [ TokenFile: <string> ]
    [ Datacenter: <string> ]
    [ Namespace: <string> ]
    Session:
       [ CheckInterval: <duration> | 10s ]
    TLS:
      [ CAFile: <string> ]
      [ CertFile: <string> ]
      [ KeyFile: <string> ]
```

### TransformConfig

```yaml
  Transform:
    Encoding:
      [ Base64: <boolean> | default = false ]
    Encryption:
      AES:
        [ Key: <string> | default = "" ]
```

### HTTPConfig

```yaml
  BindAddress: <string>
  TLS:
    CertFile: <string>
    KeyFile: <string>
  Transform: <TransformConfig>
  MaxBodyBytes: <int64>
```

### KafkaConfig

```yaml
  Kafka:
    Brokers:
      - <string>
    Key: <string>
    Secret: <string>
    [ Group: <string> | default = "" ]
    Topic: <string>
    TLS: <boolean>
    AutoCommit: <boolean>
    Transform: <TransformConfig>
```

### StorageConfig

```yaml
  Filesystem:
    [ Directory: <strong> | default = "" ]
  Encryption:
    AES:
      [ Base64Key: <string> | default = "" ]
    Encoding: <string> # Example: base64
```

### Events
```yaml
  Events:
    Stream:
      Kafka: <KafkaConfig>
    Webhook:
      [ Endpoint: <string> | default = "" ]
```


### ODFI

```yaml
  ODFI:
    Origination:
      Kafka: <KafkaConfig>
      Merging:
        Storage: <StorageConfig>
    Listen:
      Processors:
        Corrections:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "CORRECTION_"
          [ PathMatcher: <string> | default = "" ]
        Reconciliation:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "RECON_"
          [ PathMatcher: <string> | default = "" ]
        Returns:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "RET_"
          [ PathMatcher: <string> | default = "" ]
      Interval: <duration>
      ShardNames:
        - <string>
      Storage:
        Directory: <string>
        [ CleanupLocalDirectory: <boolean> | default = false]
        [ KeepRemoteFiles: <boolean> | default = false]
        [ RemoveZeroByteFiles: <boolean> | default = false]
      Events: <Events>
```

### RDFI

```yaml
  RDFI:
    Receive:
      ShardNames:
        - <string>
      Events: <Events>
    Reply:
      Processors:
        Incoming:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "CORRECTION_"
          [ PathMatcher: <string> | default = "" ]
        Prenotes:
          [ Enabled: <boolean> | default = false]
          # Partial filename to match on. Example: "PRENOTE_"
          [ PathMatcher: <string> | default = "" ]
      Kafka: <KafkaConfig>
```

### File Agents
```yaml
  FileAgents:
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
      # Base filepath on remote server
      Path: <string>
      Notifications:
        Email:
          - <string>
        PagerDuty:
          - <string>
        Slack:
          - <string>
      AllowedIPs: <string>
    Retry:
      Interval: <duration>
      MaxRetries: <integer>
    DefaultAgentID: <string>
```

### AuditTrail

```yaml
  # BucketURI is the S3-compatiable bucket location. AWS S3 and Google Cloud Storage are supported.
  # See https://gocloud.dev/howto/blob/ for more information on configuring each cloud provide.r.
  # Example: s3://my-bucket?region=us-west-1 OR gcs://my-bucket/
  BucketURI: <string>
  GPG:
    KeyFile: <string>
      Signer:
        KeyFile: <string>
        KeyPassword: <string>
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
        MergeSettings:
          # If Conditions is nil files are merged until reaching Nacha's limit of 10,000 lines
          Conditions:
            MaxLines: <integer>
            MaxDollarAmount: <integer>
          FlattenBatches: {} # Specify a non-null object to flatten batches
        OutboundFilenameTemplate: <string>
        Audit: <AuditTrail>
        Output:
          Format: <string> # Example nacha, base64, encrypted-bytes
        Notifications:
          Email:
            - <string>
          PagerDuty:
            - <string>
          Slack:
            - <string>
    Mappings:
      <string>: <string>
    Default: <string>
```

### Errors
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

### Notifications

```yaml
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
