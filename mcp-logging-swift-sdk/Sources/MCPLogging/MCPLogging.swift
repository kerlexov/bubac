import Foundation

/// Main module for MCP Logging Swift SDK
/// 
/// This module provides centralized logging capabilities for Swift applications
/// across iOS, macOS, watchOS, and tvOS platforms. It automatically captures
/// application logs and forwards them to a centralized MCP logging server.
///
/// Key Features:
/// - Structured logging with metadata support
/// - Automatic device information collection
/// - Local buffering with rotation strategy
/// - Network resilience with retry logic and circuit breaker
/// - System integration with OSLog
/// - App lifecycle event capture
/// - Crash reporting and exception handling

/// Current version of the SDK
public let MCPLoggingVersion = "1.0.0"

/// Shared logger instance for convenience
private var sharedLoggerInstance: MCPLogger?
private let sharedLoggerQueue = DispatchQueue(label: "com.mcplogging.shared")

/// Provides access to a shared MCP logger instance
public class MCPLogging {
    
    /// Configures and returns a shared logger instance
    /// - Parameter config: Logger configuration
    /// - Returns: Configured MCP logger instance
    /// - Throws: Configuration errors
    public static func configure(with config: LoggerConfig) throws -> MCPLogger {
        return try sharedLoggerQueue.sync {
            if let existing = sharedLoggerInstance {
                return existing
            }
            
            let logger = try MCPLogger(config: config)
            sharedLoggerInstance = logger
            return logger
        }
    }
    
    /// Returns the shared logger instance if configured
    public static var shared: MCPLogger? {
        return sharedLoggerQueue.sync {
            return sharedLoggerInstance
        }
    }
    
    /// Removes the shared logger instance
    public static func reset() {
        sharedLoggerQueue.sync(flags: .barrier) {
            sharedLoggerInstance = nil
        }
    }
}

/// Convenience logging functions using shared logger
public extension MCPLogging {
    
    /// Logs a debug message using the shared logger
    static func debug(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        shared?.debug(message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Logs an info message using the shared logger
    static func info(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        shared?.info(message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Logs a warning message using the shared logger
    static func warn(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        shared?.warn(message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Logs an error message using the shared logger
    static func error(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        shared?.error(message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Logs a fatal message using the shared logger
    static func fatal(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        shared?.fatal(message, metadata: metadata, file: file, function: function, line: line)
    }
}