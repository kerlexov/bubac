# MCP Logging Go SDK

A Go SDK for centralized logging with MCP (Model Context Protocol) integration. This SDK provides automatic log capture and forwarding to the centralized MCP logging server with built-in resilience features.

## Features

- **Structured Logging**: Support for structured log entries with metadata
- **Multiple Log Levels**: Debug, Info, Warn, Error, and Fatal levels
- **Automatic Buffering**: Local buffering with configurable size and flush intervals
- **Resilience**: Exponential backoff retry logic and circuit breaker pattern
- **Source Location Tracking**: Automatic capture of file, line, and function information
- **Context Support**: Context-aware logging methods
- **Library Adapters**: Integration with popular Go logging libraries (log, logrus, zap)
- **Non-blocking**: All logging operations are non-blocking

## Installation

```bash
go get github.com/kerlexov/mcp-logging-go-sdk
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/kerlexov/mcp-logging-go-sdk/pkg/logger"
)

func main() {
    // Create configuration
    config := logger.DefaultConfig()
    config.ServiceName = "my-service"
    config.AgentID = "agent-001"
    config.ServerURL = "http://localhost:9080"
    
    // Create logger
    mcpLogger, err := logger.New(config)
    if err != nil {
        log.Fatalf("Failed to create logger: %v", err)
    }
    defer mcpLogger.Close()
    
    // Basic logging
    mcpLogger.Info("Application started")
    mcpLogger.Debug("Debug information")
    mcpLogger.Warn("Warning message")
    mcpLogger.Error("Error occurred")
    
    // Logging with metadata
    mcpLogger.Info("User action", 
        logger.Field{Key: "user_id", Value: "123"},
        logger.Field{Key: "action", Value: "login"},
    )
}
```

### Configuration

```go
config := logger.Config{
    ServerURL:     "http://localhost:9080",  // MCP server URL
    ServiceName:   "my-service",             // Service identifier
    AgentID:       "agent-001",              // Agent identifier
    BufferSize:    1000,                     // Local buffer size
    FlushInterval: 5 * time.Second,          // Flush interval
    HTTPTimeout:   10 * time.Second,         // HTTP timeout
    EnableHealthCheck: true,                 // Enable health checks
    HealthCheckInterval: 30 * time.Second,   // Health check interval
    MaxRetries:    3,                        // Max retry attempts
    RetryConfig: logger.RetryConfig{
        InitialInterval: 1 * time.Second,
        MaxInterval:     30 * time.Second,
        MaxElapsedTime:  5 * time.Minute,
        Multiplier:      2.0,
        RandomizationFactor: 0.1,
    },
}
```

### Context Logging

```go
ctx := context.Background()

mcpLogger.InfoContext(ctx, "Processing request",
    logger.Field{Key: "request_id", Value: "req-123"},
)

mcpLogger.ErrorContext(ctx, "Request failed",
    logger.Field{Key: "error", Value: err.Error()},
)
```

### Logger with Fields

```go
// Create a logger with default fields
contextLogger := mcpLogger.WithFields(
    logger.Field{Key: "module", Value: "auth"},
    logger.Field{Key: "version", Value: "1.0.0"},
)

contextLogger.Info("Authentication successful",
    logger.Field{Key: "user_id", Value: "user123"},
)

// Override service name or agent ID
serviceLogger := mcpLogger.WithServiceName("auth-service")
agentLogger := mcpLogger.WithAgentID("auth-agent-001")
```

## Library Adapters

### Standard Log Adapter

```go
import "github.com/kerlexov/mcp-logging-go-sdk/pkg/adapters"

// Redirect standard log to MCP logger
adapter := adapters.NewStandardLogAdapter(mcpLogger)
log.Println("This will go to MCP logger")
```

### Logrus Adapter

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/kerlexov/mcp-logging-go-sdk/pkg/adapters"
)

// Install logrus hook
adapters.InstallLogrusHook(mcpLogger)

// Now all logrus logs will be forwarded to MCP
logrus.Info("This goes to both logrus and MCP")
```

### Zap Adapter

```go
import (
    "go.uber.org/zap"
    "github.com/kerlexov/mcp-logging-go-sdk/pkg/adapters"
)

// Create zap logger with MCP core
zapLogger := adapters.NewZapLogger(mcpLogger)
zapLogger.Info("This goes to MCP via zap")

// Or use sugared logger
sugar := adapters.NewZapSugaredLogger(mcpLogger)
sugar.Infow("User action", "user_id", 123, "action", "login")
```

## Error Handling

The SDK implements several resilience patterns:

### Circuit Breaker
- Opens after 5 consecutive failures
- Stays open for 60 seconds
- Allows limited requests in half-open state

### Exponential Backoff
- Starts with 1-second delay
- Increases by 2x multiplier with 10% jitter
- Maximum delay of 30 seconds
- Total retry time limit of 5 minutes

### Buffer Management
- Configurable buffer size (default: 1000 entries)
- Automatic rotation when buffer is full
- Non-blocking log operations

## Testing

Run the tests:

```bash
cd mcp-logging-go-sdk
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## Examples

See the [examples](./examples/) directory for more detailed usage examples:

- [Basic Usage](./examples/basic/main.go) - Simple logging example
- [Advanced Configuration](./examples/advanced/main.go) - Custom configuration
- [Library Integration](./examples/integration/main.go) - Integration with other libraries

## License

MIT License