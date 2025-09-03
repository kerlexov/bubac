# MCP Logging Server

A centralized logging system that provides log collection and retrieval for AI agents across multiple platforms through an MCP (Model Context Protocol) interface.

## Overview

The MCP Logging Server is designed to collect logs from distributed AI agent services and make them available for debugging and monitoring through a standardized MCP interface. It supports multiple platforms including Go, Swift, Express.js, React, React Native, and Kotlin.

## Features

- **Centralized Log Collection**: Collect logs from multiple services and platforms
- **MCP Interface**: Query logs through Model Context Protocol tools
- **Multi-Platform SDKs**: Native logging libraries for various platforms
- **Configurable Retention**: Automatic log cleanup based on age and level
- **Search & Filtering**: Full-text search and advanced filtering capabilities
- **High Performance**: Efficient buffering and batch processing
- **Resilient**: Built-in retry logic and error handling

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client SDKs   │───▶│  Log Ingestion  │───▶│   Log Storage   │
│  (Go, Swift,    │    │     Server      │    │   & Indexing    │
│   React, etc.)  │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   MCP Client    │◀───│   MCP Server    │
                       │   (AI Agent)    │    │   Interface     │
                       └─────────────────┘    └─────────────────┘
```

## Quick Start

### 1. Installation

```bash
git clone https://github.com/kerlexov/mcp-logging-server
cd mcp-logging-server
go mod download
```

### 2. Configuration

Copy the example configuration:

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` to match your environment:

```yaml
server:
  ingestion_port: 8080
  mcp_port: 8081

storage:
  type: sqlite
  connection_string: "./logs.db"
  max_connections: 10

retention:
  default_days: 30
  by_level:
    DEBUG: 7
    INFO: 30
    WARN: 90
    ERROR: 365
    FATAL: 365
```

### 3. Run the Server

```bash
go run cmd/server/main.go
```

The server will start two services:
- **Log Ingestion API**: `http://localhost:8080` - Receives logs from SDKs
- **MCP Server**: `http://localhost:8081` - Provides MCP tools for log querying

## Configuration

### Environment Variables

You can override configuration values using environment variables:

- `MCP_LOGGING_CONFIG`: Path to configuration file
- `MCP_LOGGING_INGESTION_PORT`: Log ingestion server port
- `MCP_LOGGING_MCP_PORT`: MCP server port
- `MCP_LOGGING_DB_CONNECTION`: Database connection string
- `MCP_LOGGING_DB_TYPE`: Database type (sqlite, postgres, clickhouse)

### Configuration File

The server looks for configuration files in the following order:
1. Path specified in `MCP_LOGGING_CONFIG` environment variable
2. `./config.yaml`
3. `./config.yml`
4. `/etc/mcp-logging/config.yaml`
5. `~/.mcp-logging/config.yaml`

## MCP Tools

The server exposes the following MCP tools:

### `query_logs`
Query logs with filtering and pagination support.

**Parameters:**
- `service_name` (string): Filter by service name
- `agent_id` (string): Filter by agent ID
- `level` (string): Filter by log level (DEBUG, INFO, WARN, ERROR, FATAL)
- `start_time` (datetime): Start of time range
- `end_time` (datetime): End of time range
- `message_contains` (string): Search in log messages
- `limit` (integer): Maximum number of results (default: 100)
- `offset` (integer): Pagination offset (default: 0)

### `get_log_details`
Retrieve specific log entries by ID.

**Parameters:**
- `ids` (array): Array of log entry IDs

### `get_service_status`
Check health and status of logging services.

### `list_services`
Get list of available services and agents.

## Data Models

### Log Entry Structure

```json
{
  "id": "uuid-v4",
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "message": "Log message content",
  "service_name": "user-service",
  "agent_id": "agent-001",
  "platform": "go",
  "metadata": {
    "user_id": "123",
    "request_id": "req-456"
  },
  "device_info": {
    "platform": "Server",
    "version": "1.21",
    "app_version": "1.2.3"
  },
  "source_location": {
    "file": "main.go",
    "line": 42,
    "function": "handleRequest"
  }
}
```

## Development

### Building

```bash
go build -o bin/mcp-logging-server cmd/server/main.go
```

### Testing

```bash
go test ./...
```

### Docker

```bash
docker build -t mcp-logging-server .
docker run -p 8080:8080 -p 8081:8081 mcp-logging-server
```

## SDKs

The following SDKs are available for different platforms:

- **Go SDK**: `github.com/kerlexov/mcp-logging-go`
- **Swift SDK**: Swift Package Manager compatible
- **Express.js SDK**: `@kerlexov/mcp-logging-express`
- **React SDK**: `@kerlexov/mcp-logging-react`
- **React Native SDK**: `@kerlexov/mcp-logging-react-native`
- **Kotlin SDK**: Maven/Gradle compatible

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

For issues and questions:
- GitHub Issues: https://github.com/kerlexov/mcp-logging-server/issues
- Documentation: https://docs.kerlexov.com/mcp-logging-server