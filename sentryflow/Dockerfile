# SPDX-License-Identifier: Apache-2.0
# Copyright 2024 Authors of SentryFlow

# --------------------------
# 1️⃣ Build Stage
# --------------------------
FROM golang:1.24.1-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

# Set working directory
WORKDIR /app

RUN apk add --no-cache make


# Copy the source code
RUN mkdir -p /protobuf/golang 
COPY protobuf/golang /protobuf/golang
COPY sentryflow/cmd cmd/
COPY sentryflow/pkg pkg/
COPY sentryflow/go.mod .
COPY sentryflow/go.sum .
COPY sentryflow/main.go main.go
COPY sentryflow/Makefile Makefile


# Copy go.mod and go.sum first for caching
RUN go mod download

# Build binary (disable CGO for static binary)
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} make build

# --------------------------
# 2️⃣ Runtime Stage
# --------------------------
FROM alpine:3.20

# Add a non-root user
RUN adduser -D -g '' sentryflow
USER sentryflow

# Copy binary from builder
COPY --from=builder /app/bin/sentryflow /usr/local/bin/sentryflow

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/sentryflow"]
