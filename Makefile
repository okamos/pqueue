NAME := pqueue
VERSION := 0.0.1
REVISION := $(shell git rev-parse --short HEAD)
LDFLAGS := -ldflags="-s -w -X \"main.version=$(VERSION)\" -X \"main.revision=$(REVISION)\" -extldflags \"-static\""
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED := 0

## Install dependencies
setup:
	dep version > /dev/null || curl -s https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	dep ensure

## Run tests
test: setup
	go test -race -v ./...

## Lint
lint:
	go get github.com/golang/lint/golint
	golint -set_exit_status $$(go list ./... | grep -v /vendor/)

## Check dead codes
vet:
	go vet ./...

## Format source codes
fmt: dev-deps
	goimports -w ./

## Install development dependencies
dev-deps:
	go get golang.org/x/tools/cmd/goimports

## Run docker containers
docker-start:
	docker run -d --rm --name pqueue-psql -p "5432:5432" postgres:10.2-alpine
	./script/wait_for_psql.sh
	psql -h localhost -U postgres -e < data/schema/job.sql

## Stop docker container
docker-stop:
	docker stop pqueue-psql

.PHONY: setup test lint vet fmt dev docker help
