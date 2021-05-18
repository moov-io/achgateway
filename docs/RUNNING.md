<!-- generated-from:276563301e0ac3cc464835f2e4747fe3042483aa7e30931de7b7c5e5ed742c94 DO NOT REMOVE, DO UPDATE -->
# ACH Gateway
**[Purpose](README.md)** | **[Configuration](CONFIGURATION.md)** | **Running** | **[Client](../pkg/client/README.md)**

---

## Running

### Getting Started

More tutorials to come on how to use this as other pieces required to handle authorization are in place!

- [Using docker-compose](#local-development)
- [Using our Docker image](#docker-image)

No configuration is required to serve on `:8200` and metrics at `:8201/metrics` in Prometheus format.

### Docker image

You can download [our docker image `moov/achgateway`](https://hub.docker.com/r/moov/achgateway/) from Docker Hub or use this repository.

### Local Development

```
make run
```

---
**[Next - Client](../pkg/client/README.md)**
