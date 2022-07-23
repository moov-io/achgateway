# generated-from:a1ae955dd09d1cf3d9d820d45ef6354f287ba25a14bff6f3643b446d7825f2b8 DO NOT REMOVE, DO UPDATE

FROM golang:1.18 as builder
WORKDIR /src
COPY go.mod .
COPY go.sum .
COPY . .

ENV HTTP_PORT=8484
ENV HEALTH_PORT=9494

EXPOSE ${HTTP_PORT}/tcp
EXPOSE ${HEALTH_PORT}/tcp

RUN ["go", "install", "github.com/githubnemo/CompileDaemon@latest"]
ENTRYPOINT CompileDaemon -directory="." -log-prefix=false -build="go build -mod vendor -o bin/main cmd/achgateway/main.go" -command="bin/main"
