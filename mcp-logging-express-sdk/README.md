# MCP Logging Express SDK

Express.js SDK for the MCP Logging Server - provides middleware and direct logging capabilities for Node.js applications.

## Installation

```bash
npm install @mcp-logging/express-sdk
```

## Quick Start

### Basic Usage with Middleware

```javascript
const express = require('express');
const { createMiddleware } = require('@mcp-logging/express-sdk');

const app = express();

// Add MCP logging middleware
app.use(createMiddleware({
  serverUrl: 'http://localhost:8080',
  serviceName: 'my-api',
  agentId: 'api-001',
  logRequests: true,
  logResponses: true
}));

app.get('/api/users', (req, res) => {
  res.json({ users: [] });
});

app.listen(3000);
```

### Direct Logging API

```javascript
const { MCPLogger } = require('@mcp-logging/express-sdk');

const logger = new MCPLogger({
  serverUrl: 'http://localhost:8080',
  serviceName: 'my-service',
  agentId: 'service-001'
});

// Log messages with different levels
logger.debug('Debug information', { userId: '123' });
logger.info('User logged in', { userId: '123', ip: '192.168.1.1' });
logger.warn('Rate limit approaching', { userId: '123', requests: 95 });
logger.error('Database connection failed', { error: 'Connection timeout' });
logger.fatal('Service shutting down', { reason: 'Critical error' });

// Graceful shutdown
process.on('SIGTERM', async () => {
  await logger.close();
  process.exit(0);
});
```

## Configuration Options

### Middleware Options

```javascript
const middlewareOptions = {
  // Required
  serverUrl: 'http://localhost:8080',
  serviceName: 'my-service',
  agentId: 'service-001',
  
  // Optional logging behavior
  logRequests: true,           // Log incoming requests
  logResponses: true,          // Log outgoing responses
  includeHeaders: false,       // Include request/response headers
  includeBody: false,          // Include request body in logs
  excludePaths: ['/health'],   // Paths to exclude from logging
  sensitiveHeaders: [          // Headers to redact
    'authorization',
    'cookie',
    'x-api-key'
  ],
  
  // Optional SDK configuration
  bufferSize: 1000,           // Local log buffer size
  flushInterval: 5000,        // Flush interval in ms
  retryConfig: {
    maxRetries: 3,
    initialDelay: 1000,
    maxDelay: 30000,
    backoffMultiplier: 2
  }
};
```

### Logger Configuration

```javascript
const loggerConfig = {
  // Required
  serverUrl: 'http://localhost:8080',
  serviceName: 'my-service',
  agentId: 'service-001',
  
  // Optional
  bufferSize: 1000,           // Local buffer size
  flushInterval: 5000,        // Auto-flush interval
  retryConfig: {              // Retry configuration
    maxRetries: 3,
    initialDelay: 1000,
    maxDelay: 30000,
    backoffMultiplier: 2
  },
  enableHealthCheck: false,   // Enable health check endpoint
  healthCheckPort: 3001       // Health check port
};
```

## Integration with Popular Logging Libraries

### Winston Integration

```javascript
const winston = require('winston');
const { createWinstonTransport } = require('@mcp-logging/express-sdk');

const logger = winston.createLogger({
  transports: [
    new winston.transports.Console(),
    createWinstonTransport({
      serverUrl: 'http://localhost:8080',
      serviceName: 'my-service',
      agentId: 'service-001'
    })
  ]
});

logger.info('This will be sent to both console and MCP server');
```

### Bunyan Integration

```javascript
const bunyan = require('bunyan');
const { createBunyanStream } = require('@mcp-logging/express-sdk');

const logger = bunyan.createLogger({
  name: 'my-service',
  streams: [
    {
      level: 'info',
      stream: process.stdout
    },
    {
      level: 'info',
      type: 'raw',
      stream: createBunyanStream({
        serverUrl: 'http://localhost:8080',
        serviceName: 'my-service',
        agentId: 'service-001'
      })
    }
  ]
});

logger.info('This will be sent to both stdout and MCP server');
```

## Advanced Usage

### Custom Error Handling

```javascript
const express = require('express');
const { createMiddleware, MCPLogger } = require('@mcp-logging/express-sdk');

const app = express();
const logger = new MCPLogger({
  serverUrl: 'http://localhost:8080',
  serviceName: 'my-api',
  agentId: 'api-001'
});

app.use(createMiddleware({
  serverUrl: 'http://localhost:8080',
  serviceName: 'my-api',
  agentId: 'api-001'
}));

// Custom error handler
app.use((err, req, res, next) => {
  logger.error('Unhandled error', {
    error: {
      name: err.name,
      message: err.message,
      stack: err.stack
    },
    request: {
      method: req.method,
      url: req.url,
      headers: req.headers
    }
  });
  
  res.status(500).json({ error: 'Internal server error' });
});
```

### Structured Logging with Context

```javascript
const { MCPLogger } = require('@mcp-logging/express-sdk');

class UserService {
  constructor() {
    this.logger = new MCPLogger({
      serverUrl: 'http://localhost:8080',
      serviceName: 'user-service',
      agentId: 'user-001'
    });
  }
  
  async createUser(userData) {
    const requestId = generateRequestId();
    
    this.logger.info('Creating user', {
      requestId,
      operation: 'createUser',
      userId: userData.id
    });
    
    try {
      const user = await this.database.create(userData);
      
      this.logger.info('User created successfully', {
        requestId,
        operation: 'createUser',
        userId: user.id,
        duration: Date.now() - startTime
      });
      
      return user;
    } catch (error) {
      this.logger.error('Failed to create user', {
        requestId,
        operation: 'createUser',
        error: {
          name: error.name,
          message: error.message,
          stack: error.stack
        },
        userData: { ...userData, password: '[REDACTED]' }
      });
      
      throw error;
    }
  }
}
```

## Features

- **Non-blocking**: All logging operations are asynchronous and won't block your application
- **Resilient**: Built-in retry logic with exponential backoff
- **Buffered**: Local buffering prevents log loss during network issues
- **High-Throughput**: Advanced buffer management for high-volume logging scenarios
- **Express Integration**: Automatic request/response logging middleware with rich metadata
- **Library Integration**: Works with Winston and Bunyan
- **Structured Logging**: Rich metadata support with automatic system information
- **Security**: Automatic sanitization of sensitive headers
- **Performance**: Minimal overhead with efficient batching
- **Error Capture**: Automatic capture of uncaught exceptions and unhandled rejections
- **Health Monitoring**: Built-in health check endpoints and system monitoring
- **Memory Management**: Intelligent buffer rotation and overflow protection

## Node.js Specific Features

### Global Error Capture

```javascript
const { MCPLogger, ErrorHandler } = require('@mcp-logging/express-sdk');

const logger = new MCPLogger(config);
const errorHandler = new ErrorHandler(logger);

// Enable automatic capture of uncaught exceptions and unhandled rejections
errorHandler.enableGlobalErrorCapture();

// Disable when shutting down
process.on('SIGTERM', () => {
  errorHandler.disableGlobalErrorCapture();
});
```

### Health Check Endpoints

```javascript
const express = require('express');
const { MCPLogger } = require('@mcp-logging/express-sdk');

const app = express();
const logger = new MCPLogger(config);
const healthChecker = logger.getHealthChecker();

// Basic health check
app.get('/health', healthChecker.createHealthEndpoint());

// Detailed health check with system information
app.get('/health/detailed', healthChecker.createDetailedHealthEndpoint());
```

### High-Throughput Buffer Management

```javascript
const logger = new MCPLogger({
  serverUrl: 'http://localhost:8080',
  serviceName: 'high-volume-service',
  agentId: 'service-001',
  bufferSize: 2000 // Automatically enables high-throughput mode
});

// Get buffer statistics
const stats = logger.getBufferStats();
console.log('Buffer stats:', stats);
```

### Enhanced Request/Response Logging

The middleware now captures extensive metadata including:

- System information (hostname, platform, Node.js version)
- Connection details (remote/local addresses and ports)
- Performance metrics (memory usage, CPU usage)
- Request/response headers and body (configurable)
- HTTP protocol details

```javascript
app.use(createMiddleware({
  serverUrl: 'http://localhost:8080',
  serviceName: 'api-service',
  agentId: 'api-001',
  includeHeaders: true,
  includeBody: true,
  logRequests: true,
  logResponses: true
}));
```

## API Reference

### MCPLogger

#### Methods

- `debug(message, metadata?)` - Log debug message
- `info(message, metadata?)` - Log info message  
- `warn(message, metadata?)` - Log warning message
- `error(message, metadata?)` - Log error message
- `fatal(message, metadata?)` - Log fatal message
- `close()` - Gracefully close logger and flush remaining logs
- `getHealthChecker()` - Get health checker instance
- `getBufferStats()` - Get current buffer statistics

### ErrorHandler

#### Methods

- `enableGlobalErrorCapture()` - Enable automatic error capture
- `disableGlobalErrorCapture()` - Disable automatic error capture

### HealthChecker

#### Methods

- `getHealthStatus()` - Get current health status
- `createHealthEndpoint()` - Create Express health check endpoint
- `createDetailedHealthEndpoint()` - Create detailed health check endpoint
- `incrementErrorCount()` - Manually increment error count
- `resetErrorCount()` - Reset error count

### createMiddleware(options)

Creates Express middleware for automatic request/response logging.

### createWinstonTransport(config)

Creates Winston transport for MCP logging.

### createBunyanStream(config)

Creates Bunyan stream for MCP logging.

## License

MIT