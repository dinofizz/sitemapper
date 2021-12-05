.DEFAULT_GOAL := build

fmt:
	go fmt ./...
.PHONY:fmt

lint: fmt
	golangci-lint run
.PHONY:lint

vet: fmt
	go vet ./...
.PHONY:vet

build: vet lint
	go build
.PHONY:build

test: build
	go test -v -cover ./...
.PHONY:test

startsite:
	caddy start ./testsite/

stopsite:
	caddy stop ./testsite/
