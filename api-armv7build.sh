#!/usr/bin/env bash
set -e

echo $IMAGE

export DOCKER_CLI_EXPERIMENTAL=enabled

docker buildx build --platform=linux/arm/v7 -f dockerfiles/dockerfile-api .
if $PUSH_IMAGE; then
  docker buildx build -t $IMAGE --platform=linux/arm/v7 -f dockerfiles/dockerfile-api --output=type=image,push=true,registry.insecure=true .
fi
