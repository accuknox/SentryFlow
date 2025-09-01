# SPDX-License-Identifier: Apache-2.0
# Copyright 2024 Authors of SentryFlow

FROM golang:1.24.1 AS builder

ARG TARGETOS
ARG TARGETARCH

# Create protobuf dir
RUN mkdir -p /protobuf/golang

# Copy protobuf files
COPY protobuf/golang /protobuf/golang

# Required to embed build info into binary.
COPY .git /.git

# Set workdir to project root
WORKDIR /sentryflow

# Copy go modules
COPY sentryflow/go.mod .
COPY sentryflow/go.sum .
RUN go mod download

# Copy sources
COPY sentryflow/cmd cmd/
COPY sentryflow/pkg pkg/
COPY sentryflow/main.go main.go
COPY sentryflow/Makefile Makefile

# Build binary using Makefile
WORKDIR /
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} make -C sentryflow build

# Final image
FROM scratch
COPY --from=builder /sentryflow/bin/sentryflow /
ENTRYPOINT ["/sentryflow"]
