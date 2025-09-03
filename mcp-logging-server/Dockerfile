# Build stage
FROM golang:1.23-bookworm AS builder

# Build arguments for cross-platform support
ARG TARGETARCH=amd64
ARG TARGETOS=linux

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set CGO environment for cross-platform compatibility
# Uses build arguments for flexible architecture support
ENV CGO_ENABLED=1
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

# Ensure GCC can handle the target architecture
ENV CC=gcc
ENV CXX=g++

# Add debugging info
RUN echo "Building for OS: $GOOS, Architecture: $GOARCH"

WORKDIR /app

# Create non-root user for build
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash -m appuser

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Copy source code
COPY . .

# Build the application with security flags
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o bin/mcp-logging-server \
    cmd/server/main.go

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    sqlite3 \
    tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash -m appuser

# Create application directory
WORKDIR /app

# Create directories with proper permissions
RUN mkdir -p /app/data /app/config /app/recovery && \
    chown -R appuser:appgroup /app

# Copy the binary and config from builder stage
COPY --from=builder --chown=appuser:appgroup /app/bin/mcp-logging-server /app/
COPY --from=builder --chown=appuser:appgroup /app/config.yaml /app/config/

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 9080 8081

# Set environment variables
ENV MCP_LOGGING_DB_CONNECTION=/app/data/logs.db \
    MCP_LOGGING_CONFIG_PATH=/app/config/config.yaml \
    MCP_LOGGING_RECOVERY_DIR=/app/recovery

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9080/health || exit 1

# Run the application
CMD ["./mcp-logging-server"]