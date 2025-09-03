# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o bin/mcp-logging-server cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/bin/mcp-logging-server .
COPY --from=builder /app/config.yaml .

# Create directories for data
RUN mkdir -p /data

# Expose ports
EXPOSE 8080 8081

# Set environment variables
ENV MCP_LOGGING_DB_CONNECTION=/data/logs.db

# Run the application
CMD ["./mcp-logging-server"]