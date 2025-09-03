# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies and security updates
RUN apk update && apk add --no-cache \
    gcc \
    musl-dev \
    sqlite-dev \
    ca-certificates \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Create non-root user for build
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

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
FROM alpine:latest

# Install runtime dependencies and security updates
RUN apk update && apk add --no-cache \
    ca-certificates \
    sqlite \
    tzdata \
    && rm -rf /var/cache/apk/* \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

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
EXPOSE 8080 8081

# Set environment variables
ENV MCP_LOGGING_DB_CONNECTION=/app/data/logs.db \
    MCP_LOGGING_CONFIG_PATH=/app/config/config.yaml \
    MCP_LOGGING_RECOVERY_DIR=/app/recovery

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./mcp-logging-server"]