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

build-standalone: vet lint
	go build -o sm ./sitemapper/cmd/standalone
.PHONY:build

build-crawlmanager: vet lint
	go build -o cm ./sitemapper/cmd/crawlmanager
.PHONY:build-crawlmanager

build-api: vet lint
	go build -o api ./sitemapper/cmd/api
.PHONY:build-api

docker-crawlmanager: build-crawlmanager
	docker build -t crawlmanager:latest -f infrastructure/dockerfile-crawlmanager .

docker-api: build-api
	docker build -t api:latest -f infrastructure/dockerfile-api .

test: build
	go test -v -cover ./...
.PHONY:test

startsite:
	caddy start ./sitemapper/testsite/

stopsite:
	caddy stop ./sitemapper/testsite/
