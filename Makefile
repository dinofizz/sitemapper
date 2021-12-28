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

docker-crawlmanager: build-crawlmanager
	docker build -t crawlmanager:latest -f infrastructure/dockerfile-crawlmanager .
.PHONY:docker-crawlmanager

docker-crawlmanager-armv7: build-crawlmanager
	docker buildx build -t 192.168.0.13:5577/crawlmanager:armv7 --platform=linux/arm/v7 -f infrastructure/dockerfile-crawlmanager --allow security.insecure --push --output=type=image,push=true,registry.insecure=true .
.PHONY:docker-crawlmanager-armv7

build-api: vet lint
	go build -o api ./sitemapper/cmd/api
.PHONY:build-api

docker-api: build-api
	docker build -t api:latest -f infrastructure/dockerfile-api .
.PHONY:docker-api

docker-api-armv7: build-api
	docker buildx build -t 192.168.0.13:5577/api:armv7 --platform=linux/arm/v7 -f infrastructure/dockerfile-api --allow security.insecure --push --output=type=image,push=true,registry.insecure=true .
.PHONY:docker-api-armv7

build-job: vet lint
	go build -o job ./sitemapper/cmd/job
.PHONY:build-job

docker-job: build-job
	docker build -t sitemapper-job:latest -f infrastructure/dockerfile-job .
.PHONY:docker-job

docker-job-armv7: build-job
	docker buildx build -t 192.168.0.13:5577/sitemapper-job:armv7 --platform=linux/arm/v7 -f infrastructure/dockerfile-job --allow security.insecure --push --output=type=image,push=true,registry.insecure=true .
.PHONY:docker-job-armv7

test: build
	go test -v -cover ./...
.PHONY:test

startsite:
	caddy start ./sitemapper/testsite/

stopsite:
	caddy stop ./sitemapper/testsite/
