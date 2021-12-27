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
	go build -o cm ./crawlmanager
.PHONY:build

docker-crawlmanager: build-crawlmanager
	docker build -t crawlmanager:latest -f infrastructure/dockerfile-crawlmanager .

test: build
	go test -v -cover ./...
.PHONY:test

startsite:
	caddy start ./sitemapper/testsite/

stopsite:
	caddy stop ./sitemapper/testsite/
