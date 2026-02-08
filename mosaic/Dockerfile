# Mosaic Add-on Dockerfile
# Build stage for Go binary
ARG BUILD_FROM=ghcr.io/home-assistant/amd64-base:3.18
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Pixlet requires these native dependencies
RUN apk add --no-cache \
    git \
    build-base \
    libwebp-dev \
    libwebp-tools \
    pango-dev \
    cairo-dev \
    pkgconf

# Copy everything needed for the build
COPY go.mod go.sum ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Resolve dependencies (go.sum may be incomplete)
RUN go mod tidy

# Build the binary with CGO enabled (required for Pixlet)
RUN go build -o mosaic ./cmd/mosaic

# Runtime stage
FROM ${BUILD_FROM}

# Install runtime dependencies (libwebp-tools includes demux/mux libs)
RUN apk add --no-cache \
    ca-certificates \
    libwebp \
    libwebp-tools \
    pango \
    cairo \
    tzdata

# Create app directories
RUN mkdir -p /app/apps /data

# Copy binary from builder
COPY --from=builder /build/mosaic /usr/bin/mosaic

# Copy web assets
COPY web/ /app/web/

# Copy bundled apps
COPY apps/ /app/apps/

# Copy bundled data (community index, etc.) - NOT to /data which is a volume mount
COPY data/ /app/data/

# Copy rootfs (s6 service scripts)
COPY rootfs/ /

# Set working directory
WORKDIR /app

# Expose port (matches MOSAIC_PORT in config.json)
EXPOSE 8176

# No CMD - s6-overlay runs the service via /etc/services.d/mosaic/run
