# MCP Logging Swift SDK

A comprehensive Swift SDK for centralized logging with MCP (Model Context Protocol) integration. This SDK provides automatic log capture and forwarding to the centralized MCP logging server with built-in resilience features for iOS, macOS, watchOS, and tvOS platforms.

## Features

- **Multi-Platform Support**: Works on iOS 12+, macOS 10.14+, watchOS 5+, and tvOS 12+
- **Structured Logging**: Support for structured log entries with metadata
- **Multiple Log Levels**: Debug, Info, Warn, Error, and Fatal levels
- **Automatic Buffering**: Local buffering with configurable size and rotation strategy
- **Network Resilience**: Exponential backoff retry logic and circuit breaker pattern
- **Device Information**: Automatic collection of device and app information
- **System Integration**: Native integration with OSLog on Apple platforms
- **Crash Reporting**: Automatic crash and exception capture
- **Lifecycle Events**: Automatic app lifecycle event logging
- **Network Monitoring**: Smart retry on network availability
- **Background Processing**: Continues logging during app backgrounding
- **Source Location**: Automatic capture of file, line, and function information

## Requirements

- **iOS**: 12.0+
- **macOS**: 10.14+
- **watchOS**: 5.0+
- **tvOS**: 12.0+
- **Swift**: 5.7+
- **Xcode**: 13.0+

## Installation

### Swift Package Manager

Add the following to your `Package.swift` file:

```swift
dependencies: [
    .package(url: "https://github.com/kerlexov/mcp-logging-swift-sdk.git", from: "1.0.0")
]
```

Or add it through Xcode:

1. File → Add Package Dependencies
2. Enter: `https://github.com/kerlexov/mcp-logging-swift-sdk.git`
3. Select version and add to target

## Quick Start

### Basic Setup

```swift
import MCPLogging

// Configure the logger
do {
    let config = try LoggerConfig.development(
        serviceName: "my-ios-app",
        agentID: "device-\(UIDevice.current.identifierForVendor?.uuidString ?? "unknown")"
    )
    
    // Create and start logger
    let logger = try MCPLogging.configure(with: config)
    logger.start()
    
    // Start logging
    logger.info("Application started")
    
} catch {
    print("Failed to setup MCP logger: \(error)")
}
```

### Using Shared Logger

```swift
import MCPLogging

// After configuration, use the shared logger anywhere
MCPLogging.info("User logged in", metadata: [
    "user_id": "12345",
    "session_id": sessionId
])

MCPLogging.error("Network request failed", metadata: [
    "url": request.url?.absoluteString ?? "",
    "status_code": response.statusCode,
    "error": error.localizedDescription
])
```

## Configuration

### Development Configuration

```swift
let config = try LoggerConfig.development(
    serverURL: URL(string: "http://localhost:8080")!, // Optional, defaults to localhost
    serviceName: "my-app",
    agentID: "unique-agent-id"
)
```

### Production Configuration

```swift
let config = try LoggerConfig.production(
    serverURL: URL(string: "https://logging.mycompany.com")!,
    serviceName: "my-app-prod",
    agentID: "prod-agent-\(deviceIdentifier)"
)
```

### Custom Configuration

```swift
let config = try LoggerConfig(
    serverURL: URL(string: "https://logs.example.com")!,
    serviceName: "my-service",
    agentID: "my-agent",
    bufferSize: 500,                    // Local buffer size
    flushInterval: 60.0,               // Flush every 60 seconds
    timeout: 15.0,                     // HTTP timeout
    maxRetries: 5,                     // Max retry attempts
    minimumLogLevel: .info,            // Only log info and above
    enableSystemIntegration: true,     // Integrate with OSLog
    enableCrashReporting: true,        // Enable crash capture
    enableLifecycleEvents: true,       // Log app lifecycle
    enableNetworkMonitoring: true,     // Monitor network changes
    retryBaseDelay: 2.0,              // Initial retry delay
    retryMaxDelay: 60.0,              // Maximum retry delay
    retryMultiplier: 2.0,             // Exponential backoff multiplier
    circuitBreakerThreshold: 3,        // Failures before circuit opens
    circuitBreakerTimeout: 30.0        // Circuit breaker timeout
)
```

## Usage Examples

### Basic Logging

```swift
import MCPLogging

let logger = MCPLogging.shared!

// Simple logging
logger.debug("Debug information")
logger.info("User action completed")
logger.warn("Deprecated API used")
logger.error("Network request failed")
logger.fatal("Critical system error")
```

### Logging with Metadata

```swift
logger.info("User authenticated", metadata: [
    "user_id": 12345,
    "username": "john_doe",
    "auth_method": "oauth",
    "timestamp": Date().timeIntervalSince1970,
    "ip_address": "192.168.1.100"
])

logger.error("Database connection failed", metadata: [
    "database": "users_db",
    "connection_string": "postgresql://...",
    "retry_count": 3,
    "error_code": "CONNECTION_TIMEOUT"
])
```

### Manual Flushing

```swift
// Force send all buffered logs
await logger.flush()

// Check buffer status
let stats = logger.bufferStats
print("Buffered logs: \(stats["count"] ?? 0)")
```

### Health Monitoring

```swift
let result = await logger.healthCheck()

switch result {
case .success(let response):
    print("Server healthy: \(response.status)")
case .failure(let error):
    print("Health check failed: \(error)")
}
```

## iOS-Specific Features

### Lifecycle Integration

The SDK automatically logs iOS app lifecycle events:

```swift
// These are logged automatically when enabled
// - Application became active
// - Application will resign active  
// - Application entered background
// - Application will enter foreground
```

### Background Processing

Logs continue to be processed when the app goes to background:

```swift
// Automatically handled - logs are flushed when app backgrounds
// Uses background tasks to complete transmission
```

### Crash Reporting

Automatic crash and exception capture:

```swift
// Uncaught exceptions and signals are automatically logged
// Crash reports include stack traces and system information
```

## macOS-Specific Features

### System Information

```swift
logger.info("System info", metadata: [
    "memory_usage": ProcessInfo.processInfo.physicalMemory,
    "processor_count": ProcessInfo.processInfo.activeProcessorCount,
    "system_version": ProcessInfo.processInfo.operatingSystemVersionString
])
```

### Process Monitoring

```swift
logger.debug("Process info", metadata: [
    "process_id": ProcessInfo.processInfo.processIdentifier,
    "process_name": ProcessInfo.processInfo.processName,
    "thermal_state": ProcessInfo.processInfo.thermalState.rawValue
])
```

## Error Handling

The SDK implements comprehensive error handling and resilience:

### Circuit Breaker

```swift
// Automatically opens circuit after configured failures
// Prevents overwhelming the server during outages
let status = logger.circuitBreakerStatus
print("Circuit state: \(status["state"] ?? "unknown")")
```

### Retry Logic

```swift
// Exponential backoff with jitter
// Automatic retry on network errors
// Configurable retry attempts and delays
```

### Local Buffering

```swift
// Logs are buffered locally when network is unavailable
// Automatic rotation when buffer is full
// Persistent across app restarts (optional)
```

## Testing

Run the test suite:

```bash
swift test
```

Run specific test cases:

```bash
swift test --filter LoggerConfigTests
swift test --filter LogBufferTests
```

## Example Apps

### iOS Demo

```bash
cd Examples/iOSDemo
open iOSDemo.xcodeproj
# Build and run on iOS device/simulator
```

### macOS Demo

```bash
cd Examples/macOSDemo  
open macOSDemo.xcodeproj
# Build and run on macOS
```

## Advanced Usage

### Custom Device Information

```swift
let customDeviceInfo = DeviceInfo(
    platform: "iOS",
    version: "16.0",
    model: "iPhone14,2",
    appVersion: "1.2.3",
    buildNumber: "456",
    deviceName: "John's iPhone",
    systemName: "iOS"
)
```

### Performance Monitoring

```swift
logger.info("Performance metrics", metadata: [
    "memory_usage": DeviceInfoCollector.memoryInfo,
    "is_simulator": DeviceInfoCollector.isSimulator,
    "platform": DeviceInfoCollector.currentPlatform
])
```

### Network Status Monitoring

```swift
// Network status changes are automatically logged
// Manual network information:
if #available(iOS 12.0, *) {
    let monitor = NetworkMonitor()
    monitor.statusUpdateHandler = { status in
        logger.info("Network status changed", metadata: [
            "status": String(describing: status)
        ])
    }
    monitor.startMonitoring()
}
```

## Troubleshooting

### Common Issues

1. **Logger not sending logs**
   ```swift
   // Check buffer stats
   let stats = logger.bufferStats
   print("Buffer count: \(stats["count"] ?? 0)")
   
   // Force flush
   await logger.flush()
   
   // Check network connectivity
   let health = await logger.healthCheck()
   ```

2. **High memory usage**
   ```swift
   // Reduce buffer size
   let config = try LoggerConfig(
       serverURL: serverURL,
       serviceName: serviceName,
       agentID: agentID,
       bufferSize: 100  // Smaller buffer
   )
   ```

3. **Missing logs**
   ```swift
   // Check minimum log level
   let config = try LoggerConfig(
       // ...
       minimumLogLevel: .debug  // Include debug logs
   )
   ```

### Debug Mode

Enable verbose debugging:

```swift
// This will log SDK internal operations to console
let config = try LoggerConfig.development(
    serviceName: "debug-app",
    agentID: "debug-agent"
)
```

## Performance Considerations

- **Asynchronous Operations**: All network operations are async and non-blocking
- **Memory Usage**: Configurable buffer sizes to control memory footprint  
- **CPU Impact**: Minimal CPU usage with efficient queuing
- **Battery Life**: Optimized for mobile battery usage
- **Network Usage**: Compressed payloads and efficient batching

## Security

- **No Sensitive Data**: Never log passwords, tokens, or personal data
- **HTTPS Only**: Production configurations should use HTTPS endpoints
- **Data Sanitization**: Metadata is automatically sanitized for JSON encoding
- **Local Storage**: Logs are kept in memory only, not persisted locally

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see LICENSE file for details

## Support

- **Issues**: [GitHub Issues](https://github.com/kerlexov/mcp-logging-swift-sdk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/kerlexov/mcp-logging-swift-sdk/discussions)
- **Documentation**: [Full Documentation](https://kerlexov.github.io/mcp-logging-swift-sdk/)