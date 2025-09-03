# MCP Logging React SDK

A React SDK for the MCP Logging Server that provides centralized logging capabilities for React applications with automatic browser console capture, error boundary integration, performance monitoring, and user interaction tracking.

## Features

- **React Context Integration** - Provider and hooks for easy integration
- **Automatic Console Capture** - Captures browser console logs automatically
- **Error Boundary Integration** - Catches and logs React component errors
- **Performance Monitoring** - Tracks Core Web Vitals and page load metrics
- **User Interaction Tracking** - Logs clicks, navigation, and scroll events
- **Local Storage Buffering** - Offline support with automatic retry
- **TypeScript Support** - Full TypeScript definitions included

## Installation

```bash
npm install @mcp-logging/react-sdk
```

## Quick Start

### 1. Wrap your app with MCPLoggerProvider

```tsx
import React from 'react';
import { MCPLoggerProvider } from '@mcp-logging/react-sdk';
import App from './App';

const config = {
  serverUrl: 'http://localhost:8080',
  serviceName: 'my-react-app',
  agentId: 'web-client-001',
};

function Root() {
  return (
    <MCPLoggerProvider config={config}>
      <App />
    </MCPLoggerProvider>
  );
}

export default Root;
```

### 2. Use the logging hook in components

```tsx
import React from 'react';
import { useMCPLogger } from '@mcp-logging/react-sdk';

function UserProfile({ userId }: { userId: string }) {
  const logger = useMCPLogger();

  const handleLogin = async () => {
    try {
      logger.info('User login attempt', { userId });
      // ... login logic
      logger.info('User login successful', { userId });
    } catch (error) {
      logger.error('User login failed', { userId, error: error.message });
    }
  };

  return (
    <button onClick={handleLogin}>
      Login
    </button>
  );
}
```

## Configuration

```tsx
interface MCPLoggerConfig {
  serverUrl: string;                    // MCP logging server URL
  serviceName: string;                  // Name of your service
  agentId: string;                      // Unique agent identifier
  bufferSize?: number;                  // Log buffer size (default: 100)
  flushInterval?: number;               // Auto-flush interval in ms (default: 5000)
  retryAttempts?: number;               // Retry attempts for failed sends (default: 3)
  retryDelay?: number;                  // Base retry delay in ms (default: 1000)
  enableConsoleCapture?: boolean;       // Capture console logs (default: true)
  enableErrorBoundary?: boolean;        // Capture unhandled errors (default: true)
  enablePerformanceMetrics?: boolean;   // Track performance metrics (default: true)
  enableUserInteractions?: boolean;     // Track user interactions (default: true)
  enableLocalStorage?: boolean;         // Use localStorage for offline buffering (default: true)
  logLevel?: LogLevel;                  // Minimum log level (default: 'INFO')
}
```

## API Reference

### Hooks

#### `useMCPLogger()`

Returns the logger instance for the current context.

```tsx
const logger = useMCPLogger();

// Log at different levels
logger.debug('Debug message', { key: 'value' });
logger.info('Info message', { key: 'value' });
logger.warn('Warning message', { key: 'value' });
logger.error('Error message', { key: 'value' });
logger.fatal('Fatal message', { key: 'value' });

// Log performance metrics
logger.logPerformance({
  pageLoadTime: 1500,
  firstContentfulPaint: 800,
});

// Log user interactions
logger.logUserInteraction({
  type: 'click',
  element: 'button#submit',
  timestamp: new Date(),
});

// Manual flush
await logger.flush();

// Check health status
const { isHealthy, lastError } = logger.getHealthStatus();
```

#### `useMCPLoggerConfig()`

Returns the current logger configuration.

```tsx
const config = useMCPLoggerConfig();
console.log(config.serviceName); // 'my-react-app'
```

### Components

#### `MCPLoggerProvider`

Context provider that initializes the logging system.

```tsx
<MCPLoggerProvider config={config}>
  <App />
</MCPLoggerProvider>
```

#### `MCPErrorBoundary`

Error boundary component that catches and logs React errors.

```tsx
import { MCPErrorBoundary, useMCPLogger } from '@mcp-logging/react-sdk';

function MyComponent() {
  const logger = useMCPLogger();
  
  return (
    <MCPErrorBoundary 
      logger={logger}
      fallback={<div>Something went wrong!</div>}
      onError={(error, errorInfo) => {
        // Custom error handling
        console.log('Error caught:', error);
      }}
    >
      <RiskyComponent />
    </MCPErrorBoundary>
  );
}
```

#### `withMCPErrorBoundary` HOC

Higher-order component for wrapping components with error boundary.

```tsx
import { withMCPErrorBoundary, useMCPLogger } from '@mcp-logging/react-sdk';

const MyComponent = () => <div>Content</div>;

const WrappedComponent = withMCPErrorBoundary(
  MyComponent,
  logger,
  <div>Error fallback</div>
);
```

## Automatic Features

### Console Capture

When enabled, the SDK automatically captures:
- `console.log()` → INFO level
- `console.warn()` → WARN level  
- `console.error()` → ERROR level

### Error Capture

Automatically captures:
- Unhandled JavaScript errors
- Unhandled promise rejections
- React component errors (when using error boundary)

### Performance Monitoring

Automatically tracks:
- Page load time
- DOM content loaded time
- First Contentful Paint (FCP)
- Largest Contentful Paint (LCP)
- Cumulative Layout Shift (CLS)
- First Input Delay (FID)

### User Interaction Tracking

Automatically logs:
- Click events (with element selector)
- Navigation events (URL changes)
- Scroll events (throttled)

## Offline Support

The SDK includes built-in offline support:
- Failed log transmissions are stored in localStorage
- Automatic retry with exponential backoff
- Logs are restored and sent when connection is restored
- Configurable buffer size limits

## TypeScript Support

The SDK is written in TypeScript and includes full type definitions:

```tsx
import { MCPLogger, LogLevel, MCPLoggerConfig } from '@mcp-logging/react-sdk';

const config: MCPLoggerConfig = {
  serverUrl: 'http://localhost:8080',
  serviceName: 'my-app',
  agentId: 'web-001',
};
```

## Examples

### Basic Usage

```tsx
import React from 'react';
import { MCPLoggerProvider, useMCPLogger } from '@mcp-logging/react-sdk';

const App = () => {
  const logger = useMCPLogger();

  React.useEffect(() => {
    logger.info('App component mounted');
  }, [logger]);

  return <div>My App</div>;
};

const Root = () => (
  <MCPLoggerProvider config={{
    serverUrl: 'http://localhost:8080',
    serviceName: 'my-app',
    agentId: 'web-001',
  }}>
    <App />
  </MCPLoggerProvider>
);
```

### With Error Boundary

```tsx
import React from 'react';
import { MCPLoggerProvider, MCPErrorBoundary, useMCPLogger } from '@mcp-logging/react-sdk';

const RiskyComponent = () => {
  const [shouldError, setShouldError] = React.useState(false);
  
  if (shouldError) {
    throw new Error('Something went wrong!');
  }
  
  return (
    <button onClick={() => setShouldError(true)}>
      Trigger Error
    </button>
  );
};

const App = () => {
  const logger = useMCPLogger();
  
  return (
    <MCPErrorBoundary logger={logger}>
      <RiskyComponent />
    </MCPErrorBoundary>
  );
};
```

### Custom Performance Tracking

```tsx
import React from 'react';
import { useMCPLogger } from '@mcp-logging/react-sdk';

const DataComponent = () => {
  const logger = useMCPLogger();

  const fetchData = async () => {
    const startTime = performance.now();
    
    try {
      const response = await fetch('/api/data');
      const data = await response.json();
      
      const endTime = performance.now();
      logger.logPerformance({
        customMetric: endTime - startTime,
      });
      
      logger.info('Data fetched successfully', { 
        recordCount: data.length,
        duration: endTime - startTime 
      });
    } catch (error) {
      logger.error('Failed to fetch data', { error: error.message });
    }
  };

  return (
    <button onClick={fetchData}>
      Fetch Data
    </button>
  );
};
```

## License

MIT