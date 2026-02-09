# SPDX-License-Identifier: Apache-2.0
# Copyright 2024 Authors of SentryFlow

REGISTRY ?= public.ecr.aws/k9v9d5v2
DOCKER_IMAGE ?= $(REGISTRY)/sentryflow-httpfilter
DOCKER_TAG ?= latest
CONTAINER_TOOL ?= docker

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

.PHONY: toolchain
toolchain: ## Install Rust WASM toolchain
	@test rustup || curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
	@rustup target add wasm32-wasip1

.PHONY: build-sidecar
build-sidecar: ## Build sidecar plugin (default)
	@cargo build --target wasm32-wasip1 --release --features sidecar

.PHONY: build-gateway
build-gateway: ## Build gateway plugin
	@cargo build --target wasm32-wasip1 --release --no-default-features --features gateway

.PHONY: build-gateway-ratelimit
build-gateway-ratelimit: ## Build gateway plugin with rate limit feature
	@cargo build --target wasm32-wasip1 --release --no-default-features --features "gateway rate-limit"

.PHONY: build
build: build-sidecar ## Build plugin (alias for build-sidecar)

.PHONY: clean
clean: ## Remove generated stuff
	@cargo clean

.PHONY: image-sidecar
image-sidecar:
	$(CONTAINER_TOOL) build --build-arg PLUGIN_TYPE=sidecar -t ${DOCKER_IMAGE}:${DOCKER_TAG}-sidecar .

.PHONY: image-gateway
image-gateway: 
	$(CONTAINER_TOOL) build --build-arg PLUGIN_TYPE=gateway -t ${DOCKER_IMAGE}:${DOCKER_TAG}-gateway .

.PHONY: image-gateway-ratelimit
image-gateway-ratelimit: 
	$(CONTAINER_TOOL) build --build-arg PLUGIN_TYPE=gateway-ratelimit -t ${DOCKER_IMAGE}:${DOCKER_TAG}-gateway-ratelimit .

.PHONY: push-sidecar
push-sidecar: ## Push sidecar container image
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE}:${DOCKER_TAG}-sidecar

.PHONY: push-gateway
push-gateway: ## Push gateway container image
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE}:${DOCKER_TAG}-gateway

.PHONY: push-gateway-ratelimit
push-gateway-ratelimit: ## Push gateway container image with rate limit feature
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE}:${DOCKER_TAG}-gateway-ratelimit

.PHONY: all
all: image-sidecar image-gateway image-gateway-ratelimit ## Build both sidecar, gateway, and gateway-ratelimit images

.PHONY: image
all: image-sidecar ## By default we make image for sidecar 

.PHONY: push-all
push-all: push-sidecar push-gateway push-gateway-ratelimit ## Push all images
