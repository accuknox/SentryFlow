# SPDX-License-Identifier: Apache-2.0
# Copyright 2024 Authors of SentryFlow

ARG PLUGIN_TYPE=sidecar

FROM rust:1.81.0 AS builder

ARG PLUGIN_TYPE=sidecar

WORKDIR /envoy-plugin

COPY . .

RUN make toolchain build

# Build the plugin with appropriate feature flag
RUN if [ "${PLUGIN_TYPE}" = "gateway" ]; then \
        cargo build --target wasm32-wasip1 --release --no-default-features --features gateway; \
    else \
        cargo build --target wasm32-wasip1 --release --features sidecar; \
    fi

FROM scratch

COPY --from=builder /envoy-plugin/target/wasm32-wasip1/release/httpfilters.wasm ./plugin.wasm
