# MCP Logging System

A comprehensive centralized logging solution with MCP (Model Context Protocol) integration for AI agents and applications. This system provides centralized log collection, storage, and retrieval through MCP tools for debugging and monitoring distributed AI services.

## Overview

The MCP Logging System consists of:

1. **Centralized Log Server** - High-performance server for log ingestion and storage
2. **MCP Server Interface** - MCP-compliant server exposing log retrieval tools
3. **Go SDK** - Native Go logging library with automatic forwarding
4. **Multi-Platform SDKs** - Planned SDKs for Swift, Express.js, React, React Native, and Kotlin

## Quick Start

### 1. Start the MCP Logging Server

The MCP Logging Server handles log ingestion and provides MCP tools for log retrieval.

#### Prerequisites

- Go 1.21 or higher
- SQLite (for default storage)

#### Installation & Setup

```bash
# Clone and navigate to server directory
cd mcp-logging-server

# Install dependencies
go mod tidy

# Build the server
go build -o bin/mcp-logging-server ./cmd/server

# Start the server with default configuration
./bin/mcp-logging-server
```

#### Configuration

Create a `config.yaml` file:

```yaml
server:
  ingestion_port: 8080      # Port for log ingestion API
  mcp_port: 8081           # Port for MCP server interface
  
storage:
  type: "sqlite"           # Options: sqlite, postgres, clickhouse
  connection_string: "./logs.db"
  max_connections: 100

retention:
  default_days: 30         # Default retention period
  by_level:
    DEBUG: 7
    INFO: 30
    WARN: 90
    ERROR: 365
    FATAL: 365

indexing:
  enabled: true
  full_text_search: true

buffer:
  size: 10000             # In-memory buffer size
  flush_interval: "5s"    # How often to flush to storage

health:
  enabled: true
  port: 8082             # Health check endpoint port
```

#### Environment Variables

```bash
# Server Configuration
export MCP_LOG_INGESTION_PORT=8080
export MCP_LOG_MCP_PORT=8081
export MCP_LOG_DB_PATH=./logs.db

# Optional: Advanced Configuration
export MCP_LOG_BUFFER_SIZE=10000
export MCP_LOG_FLUSH_INTERVAL=5s
export MCP_LOG_RETENTION_DAYS=30
```

#### Docker Setup

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY mcp-logging-server/ .
RUN go mod tidy && go build -o bin/server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bin/server .
EXPOSE 8080 8081 8082

CMD ["./server"]
```

```bash
# Build and run with Docker
docker build -t mcp-logging-server .
docker run -p 8080:8080 -p 8081:8081 -p 8082:8082 mcp-logging-server
```

#### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  mcp-logging-server:
    build: ./mcp-logging-server
    ports:
      - "8080:8080"  # Log ingestion
      - "8081:8081"  # MCP interface
      - "8082:8082"  # Health checks
    volumes:
      - ./data:/data
    environment:
      - MCP_LOG_DB_PATH=/data/logs.db
      - MCP_LOG_RETENTION_DAYS=30
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

```bash
# Start with docker-compose
docker-compose up -d
```

### 2. Integrate with Claude Desktop

Add the MCP logging server to Claude Desktop to access logs through MCP tools.

#### Claude Desktop Configuration

Edit your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "mcp-logging": {
      "command": "node",
      "args": ["/path/to/mcp-logging-server/mcp-proxy.js"],
      "env": {
        "MCP_SERVER_URL": "http://localhost:8081"
      }
    }
  }
}
```

#### MCP Proxy Script

Create `mcp-logging-server/mcp-proxy.js`:

```javascript
#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

// Path to the MCP logging server binary
const serverPath = path.join(__dirname, 'bin', 'mcp-logging-server');
const mcpPort = process.env.MCP_PORT || '8081';

// Start the MCP server
const server = spawn(serverPath, ['--mcp-only', '--port', mcpPort], {
  stdio: 'inherit',
  env: { ...process.env }
});

server.on('error', (err) => {
  console.error('Failed to start MCP logging server:', err);
  process.exit(1);
});

server.on('close', (code) => {
  console.log(`MCP logging server exited with code ${code}`);
  process.exit(code);
});

// Handle cleanup
process.on('SIGINT', () => {
  server.kill('SIGINT');
});

process.on('SIGTERM', () => {
  server.kill('SIGTERM');
});
```

Make it executable:

```bash
chmod +x mcp-logging-server/mcp-proxy.js
```

#### Available MCP Tools

Once configured, Claude will have access to these tools:

1. **`query_logs`** - Search and filter logs
   ```json
   {
     "service_name": "my-service",
     "level": "ERROR",
     "start_time": "2024-01-01T00:00:00Z",
     "end_time": "2024-01-01T23:59:59Z",
     "message_contains": "exception",
     "limit": 100
   }
   ```

2. **`get_log_details`** - Get specific log entries by ID
   ```json
   {
     "log_ids": ["log-id-1", "log-id-2"]
   }
   ```

3. **`get_service_status`** - Check logging system health
4. **`list_services`** - List available services and agents

### 3. Integrate with OpenCode

Add MCP logging capabilities to OpenCode for automatic code execution logging.

#### OpenCode Configuration

Create or edit `~/.opencode/config.json`:

```json
{
  "mcp": {
    "servers": {
      "logging": {
        "command": "/path/to/mcp-logging-server/bin/mcp-logging-server",
        "args": ["--mcp-only", "--port", "8081"],
        "env": {
          "MCP_LOG_LEVEL": "INFO"
        }
      }
    }
  },
  "logging": {
    "enabled": true,
    "server_url": "http://localhost:8080",
    "service_name": "opencode",
    "agent_id": "opencode-agent-001",
    "log_commands": true,
    "log_outputs": true,
    "log_errors": true
  }
}
```

#### OpenCode Integration Script

Create `~/.opencode/integrations/mcp-logging.js`:

```javascript
// OpenCode MCP Logging Integration
const axios = require('axios');

class MCPLoggingIntegration {
  constructor(config) {
    this.serverUrl = config.server_url;
    this.serviceName = config.service_name;
    this.agentId = config.agent_id;
    this.enabled = config.enabled || false;
  }

  async logCommand(command, metadata = {}) {
    if (!this.enabled) return;
    
    await this.sendLog('INFO', `Command executed: ${command}`, {
      type: 'command',
      command,
      ...metadata
    });
  }

  async logOutput(command, output, metadata = {}) {
    if (!this.enabled) return;
    
    await this.sendLog('DEBUG', 'Command output', {
      type: 'output',
      command,
      output: output.substring(0, 1000), // Truncate long outputs
      ...metadata
    });
  }

  async logError(command, error, metadata = {}) {
    if (!this.enabled) return;
    
    await this.sendLog('ERROR', `Command failed: ${command}`, {
      type: 'error',
      command,
      error: error.message || error,
      stack_trace: error.stack,
      ...metadata
    });
  }

  async sendLog(level, message, metadata) {
    try {
      const logEntry = {
        level,
        message,
        service_name: this.serviceName,
        agent_id: this.agentId,
        platform: 'opencode',
        metadata: {
          timestamp: new Date().toISOString(),
          ...metadata
        }
      };

      await axios.post(`${this.serverUrl}/api/logs`, {
        logs: [logEntry]
      }, {
        timeout: 5000,
        headers: { 'Content-Type': 'application/json' }
      });
    } catch (error) {
      // Silently fail to avoid disrupting OpenCode operations
      console.error('Failed to send log to MCP server:', error.message);
    }
  }
}

module.exports = MCPLoggingIntegration;
```

#### Using in OpenCode Commands

OpenCode can now automatically log all operations:

```bash
# These commands will be automatically logged
opencode "Create a new React component"
opencode "Fix the TypeScript errors in src/utils.ts"
opencode "Run the test suite and fix any failures"
```

The logs will include:
- Command execution details
- File changes made
- Test results
- Error messages and stack traces
- Execution time and context

### 4. Using the Go SDK in Your Applications

#### Installation

```bash
go get github.com/kerlexov/mcp-logging-go-sdk
```

#### Basic Usage

```go
package main

import (
    "log"
    "github.com/kerlexov/mcp-logging-go-sdk/pkg/logger"
)

func main() {
    // Configure the logger
    config := logger.DefaultConfig()
    config.ServiceName = "my-go-app"
    config.AgentID = "app-instance-001"
    config.ServerURL = "http://localhost:8080"
    
    // Create logger
    mcpLogger, err := logger.New(config)
    if err != nil {
        log.Fatalf("Failed to create MCP logger: %v", err)
    }
    defer mcpLogger.Close()
    
    // Start logging
    mcpLogger.Info("Application started")
    
    // Your application code here...
    mcpLogger.Info("Processing user request", 
        logger.Field{Key: "user_id", Value: "123"},
        logger.Field{Key: "endpoint", Value: "/api/users"},
    )
}
```

### 5. Querying Logs via Claude

Once everything is set up, you can ask Claude to help with log analysis:

**Example Queries:**

- "Show me all error logs from the last hour for the user-service"
- "What were the most recent DEBUG logs from opencode?"
- "Find logs containing 'database connection' from yesterday"
- "Show me the health status of all logging services"
- "List all services that have sent logs today"

**Claude Usage:**

```
You: Show me recent error logs from my Go application

Claude: I'll check the recent error logs for you.

*Uses query_logs tool*
{
  "service_name": "my-go-app", 
  "level": "ERROR",
  "limit": 20
}

Based on the logs, I found 3 recent errors in your Go application:
1. Database connection timeout at 14:32
2. Invalid user authentication at 14:28  
3. File not found error at 14:25

Would you like me to get more details about any of these errors?
```

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Applications  │    │  MCP Log Server  │    │     Claude      │
│                 │    │                  │    │   Desktop       │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ ┌─────────────┐ │
│ │   Go SDK    │─┼────┼─│ Log Ingestion│ │    │ │   MCP       │ │
│ └─────────────┘ │    │ │   API :8080  │ │    │ │  Client     │ │
│                 │    │ └──────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    └─────────────────┘
│ │ OpenCode    │─┼────┼─│ SQLite/PG DB │ │             │
│ │ Integration │ │    │ └──────────────┘ │             │
│ └─────────────┘ │    │ ┌──────────────┐ │             │
└─────────────────┘    │ │ MCP Server   │─┼─────────────┘
                       │ │   :8081      │ │
                       │ └──────────────┘ │
                       │ ┌──────────────┐ │
                       │ │   Health     │ │
                       │ │   :8082      │ │
                       │ └──────────────┘ │
                       └──────────────────┘
```

## Monitoring & Health Checks

### Health Check Endpoints

```bash
# Server health
curl http://localhost:8082/health

# MCP server status
curl http://localhost:8081/health

# Storage health
curl http://localhost:8082/storage/health
```

### Monitoring Logs

```bash
# View server logs
tail -f /var/log/mcp-logging-server.log

# Monitor log ingestion
curl http://localhost:8082/metrics
```

## Troubleshooting

### Common Issues

1. **Server won't start**
   ```bash
   # Check port availability
   netstat -an | grep :8080
   
   # Check configuration
   ./bin/mcp-logging-server --config-check
   ```

2. **Claude can't connect to MCP server**
   - Verify MCP proxy script path and permissions
   - Check Claude Desktop config file syntax
   - Ensure server is running on correct port

3. **Logs not appearing**
   - Check network connectivity to server
   - Verify service_name and agent_id in SDK config
   - Check server logs for ingestion errors

4. **OpenCode integration not working**
   - Verify integration script is loaded
   - Check OpenCode configuration
   - Test manual log sending with curl

### Debug Commands

```bash
# Test log ingestion
curl -X POST http://localhost:8080/api/logs \
  -H "Content-Type: application/json" \
  -d '{"logs":[{"level":"INFO","message":"test","service_name":"test","agent_id":"test","platform":"curl"}]}'

# Test MCP tools (if server supports direct testing)
curl -X POST http://localhost:8081/mcp/query_logs \
  -H "Content-Type: application/json" \
  -d '{"service_name":"test","limit":10}'
```

## Contributing

See individual component directories for development setup:

- [MCP Server Development](./mcp-logging-server/README.md)
- [Go SDK Development](./mcp-logging-go-sdk/README.md)

## License

MIT License
