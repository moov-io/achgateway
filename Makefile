
# generated-from:8ef700b33a05ab58ec9e7fd3ad1a0d8a99a742beeefc09d26bc7e4b6dd2ad699 DO NOT REMOVE, DO UPDATE

PLATFORM=$(shell uname -s | tr '[:upper:]' '[:lower:]')
PWD := $(shell pwd)

ifndef VERSION
	VERSION := $(shell git describe --tags --abbrev=0)
endif

COMMIT_HASH :=$(shell git rev-parse --short HEAD)
DEV_VERSION := dev-${COMMIT_HASH}

USERID := $(shell id -u $$USER)
GROUPID:= $(shell id -g $$USER)

export GOPRIVATE=github.com/moov-io

all: install update build

.PHONY: install
install:
	go mod tidy
	go install github.com/markbates/pkger/cmd/pkger
	go mod vendor

update:
	pkger -include /migrations -include /configs/config.default.yml
	go mod vendor

build:
	go build -mod=vendor -ldflags "-X github.com/moov-io/achgateway.Version=${VERSION}" -o bin/achgateway github.com/moov-io/achgateway/cmd/achgateway

.PHONY: setup
setup:
	docker-compose up -d --force-recreate --remove-orphans

.PHONY: check
check:
ifeq ($(OS),Windows_NT)
	@echo "Skipping checks on Windows, currently unsupported."
else
	@wget -O lint-project.sh https://raw.githubusercontent.com/moov-io/infra/master/go/lint-project.sh
	@chmod +x ./lint-project.sh
	COVER_THRESHOLD=44.5 PROFILE_GOTEST=yes ./lint-project.sh
endif

.PHONY: teardown
teardown:
	-docker-compose down --remove-orphans
	-docker-compose rm -sfv

docker: update
	docker build --pull --build-arg VERSION=${VERSION} -t moov/achgateway:${VERSION} -f Dockerfile .
	docker tag moov/achgateway:${VERSION} moov/achgateway:latest

docker-push:
	docker push moov/achgateway:${VERSION}
	docker push moov/achgateway:latest

.PHONY: dev-docker
dev-docker: update
	docker build --pull --build-arg VERSION=${DEV_VERSION} -t moov/achgateway:${DEV_VERSION} -f Dockerfile .

.PHONY: dev-push
dev-push:
	docker push moov/achgateway:${DEV_VERSION}

# Extra utilities not needed for building

run: update build
	./bin/achgateway

docker-run:
	docker run -v ${PWD}/data:/data -v ${PWD}/configs:/configs --env APP_CONFIG="/configs/config.yml" -it --rm moov-io/achgateway:${VERSION}

test: update
	go test -cover github.com/moov-io/achgateway/...

.PHONY: clean
clean:
ifeq ($(OS),Windows_NT)
	@echo "Skipping cleanup on Windows, currently unsupported."
else
	@rm -rf cover.out coverage.txt misspell* staticcheck*
	@rm -rf ./bin/
endif

# For open source projects

# From https://github.com/genuinetools/img
.PHONY: AUTHORS
AUTHORS:
	@$(file >$@,# This file lists all individuals having contributed content to the repository.)
	@$(file >>$@,# For how it is generated, see `make AUTHORS`.)
	@echo "$(shell git log --format='\n%aN <%aE>' | LC_ALL=C.UTF-8 sort -uf)" >> $@

dist: clean build
ifeq ($(OS),Windows_NT)
	CGO_ENABLED=1 GOOS=windows go build -o bin/achgateway.exe cmd/achgateway/*
else
	CGO_ENABLED=1 GOOS=$(PLATFORM) go build -o bin/achgateway-$(PLATFORM)-amd64 cmd/achgateway/*
endif
