<!-- generated-from:c55598a61b9a43ce98e8500432e597669da2f8955b40d9e772eac6011e2bb1c5 DO NOT REMOVE, DO UPDATE -->
# ACH Conductor
**[Purpose](README.md)** | **Configuration** | **[Running](RUNNING.md)** | **[Client](../pkg/client/README.md)**

---

## Configuration
Custom configuration for this application may be specified via an environment variable `APP_CONFIG` to a configuration file that will be merged with the default configuration file.

- [Default Configuration](../configs/config.default.yml)
- [Config Source Code](../pkg/service/model_config.go)
- Full Configuration
  ```yaml
  ACH Conductor:

    # Service configurations
    Servers:

      # Public service configuration
      Public:
        Bind:
          # Address and port to listen on.
          Address: ":8200"

      # Health/Admin service configuration.
      Admin:
        Bind:
          # Address and port to listen on.
          Address: ":8201"

    # All database configuration is done here. Only one connector can be configured.
    Database:

      # Database name to use for selected connector.
      DatabaseName: "identity"

      # MySql configuration
      MySQL:  
        Address: tcp(mysqlidentity:3306)
        User: identity
        Password: identity

      # OR uses the sqllite db
      SQLLite:
        Path: ":memory:"

    # Gateway configuration to look up public keys to verify JWT tokens.
    Gateway:

      # If neither http or file are specified, the service will generate random keys
      Keys:

        # Pulls Keys from endpoints
        HTTP:
        URLs:
        - http://tumbler:8204/.well-known/jwks.json

        # Pulls keys from the disk
        File:
          Paths: 
          - ./configs/gateway-jwks-sig-pub.json

  ```

---
**[Next - Running](RUNNING.md)**
