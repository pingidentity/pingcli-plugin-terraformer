# Multi-stage Dockerfile to produce a Linux-native pingcli binary

# Builder stage: compile the Go binary for the target platform
FROM golang:1.25.1-alpine AS builder
WORKDIR /src

# Install build deps
RUN apk add --no-cache git build-base

# Leverage Docker layer caching for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build for the target platform provided by BuildKit (or default to linux/amd64)
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -trimpath -o /out/pingcli-terraformer .

# Runtime stage: minimal image with CA certificates
FROM alpine:3.19
RUN apk add --no-cache ca-certificates

# Create non-root user for security
RUN addgroup -g 1000 terraformer && \
    adduser -D -u 1000 -G terraformer terraformer

# Copy the compiled binary from the builder
COPY --from=builder /out/pingcli-terraformer /usr/local/bin/pingcli-terraformer

# Create output directory with proper permissions
RUN mkdir /output && chown terraformer:terraformer /output

# Switch to non-root user
USER terraformer
WORKDIR /output

# Set the entry point
ENTRYPOINT ["pingcli-terraformer"]

# Allow subcommands
CMD []