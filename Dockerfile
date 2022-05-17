# generated-from:a1ae955dd09d1cf3d9d820d45ef6354f287ba25a14bff6f3643b446d7825f2b8 DO NOT REMOVE, DO UPDATE

FROM golang:1.18-buster as builder
WORKDIR /src
ARG VERSION

RUN apt-get update && apt-get upgrade -y && apt-get install -y make gcc g++ ca-certificates

COPY . .

RUN VERSION=${VERSION} make build

FROM debian:stable-slim AS runtime
LABEL maintainer="Moov <support@moov.io>"

WORKDIR /

RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates curl \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/bin/achgateway /app/

ENV HTTP_PORT=8484
ENV HEALTH_PORT=9494

EXPOSE ${HTTP_PORT}/tcp
EXPOSE ${HEALTH_PORT}/tcp

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
	CMD curl -f http://localhost:${HEALTH_PORT}/live || exit 1

VOLUME [ "/data", "/configs" ]

ENTRYPOINT ["/app/achgateway"]
