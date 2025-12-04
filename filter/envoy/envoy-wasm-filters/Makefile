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

.PHONY: build
build: build-sidecar ## Build plugin (alias for build-sidecar)

.PHONY: clean
clean: ## Remove generated stuff
	@cargo clean

.PHONY: image-sidecar
image-sidecar: build-sidecar ## Build sidecar container image
	$(CONTAINER_TOOL) build --build-arg PLUGIN_TYPE=sidecar -t ${DOCKER_IMAGE}:v0.4-sidecar .

.PHONY: image-gateway
image-gateway: build-gateway ## Build gateway container image
	$(CONTAINER_TOOL) build --build-arg PLUGIN_TYPE=gateway -t ${DOCKER_IMAGE}:v0.4-gateway .

.PHONY: push-sidecar
push-sidecar: ## Push sidecar container image
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE}:v0.4-sidecar

.PHONY: push-gateway
push-gateway: ## Push gateway container image
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE}:v0.4-gateway

.PHONY: all
all: image-sidecar image-gateway ## Build both sidecar and gateway images

.PHONY: push-all
push-all: push-sidecar push-gateway ## Push both images

