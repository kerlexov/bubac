import Foundation

/// Configuration for MCP Logger
public struct LoggerConfig {
    
    /// Server configuration
    public let serverURL: URL
    public let serviceName: String
    public let agentID: String
    
    /// Buffer configuration
    public let bufferSize: Int
    public let flushInterval: TimeInterval
    
    /// Network configuration
    public let timeout: TimeInterval
    public let maxRetries: Int
    
    /// Logging configuration
    public let minimumLogLevel: LogLevel
    public let enableSystemIntegration: Bool
    public let enableCrashReporting: Bool
    public let enableLifecycleEvents: Bool
    public let enableNetworkMonitoring: Bool
    
    /// Retry configuration
    public let retryBaseDelay: TimeInterval
    public let retryMaxDelay: TimeInterval
    public let retryMultiplier: Double
    
    /// Circuit breaker configuration
    public let circuitBreakerThreshold: Int
    public let circuitBreakerTimeout: TimeInterval
    
    public init(
        serverURL: URL,
        serviceName: String,
        agentID: String,
        bufferSize: Int = 1000,
        flushInterval: TimeInterval = 30.0,
        timeout: TimeInterval = 10.0,
        maxRetries: Int = 3,
        minimumLogLevel: LogLevel = .debug,
        enableSystemIntegration: Bool = true,
        enableCrashReporting: Bool = true,
        enableLifecycleEvents: Bool = true,
        enableNetworkMonitoring: Bool = true,
        retryBaseDelay: TimeInterval = 1.0,
        retryMaxDelay: TimeInterval = 30.0,
        retryMultiplier: Double = 2.0,
        circuitBreakerThreshold: Int = 5,
        circuitBreakerTimeout: TimeInterval = 60.0
    ) throws {
        guard !serviceName.isEmpty else {
            throw LoggerConfigError.invalidServiceName
        }
        
        guard !agentID.isEmpty else {
            throw LoggerConfigError.invalidAgentID
        }
        
        guard bufferSize > 0 else {
            throw LoggerConfigError.invalidBufferSize
        }
        
        guard flushInterval > 0 else {
            throw LoggerConfigError.invalidFlushInterval
        }
        
        guard timeout > 0 else {
            throw LoggerConfigError.invalidTimeout
        }
        
        self.serverURL = serverURL
        self.serviceName = serviceName
        self.agentID = agentID
        self.bufferSize = bufferSize
        self.flushInterval = flushInterval
        self.timeout = timeout
        self.maxRetries = maxRetries
        self.minimumLogLevel = minimumLogLevel
        self.enableSystemIntegration = enableSystemIntegration
        self.enableCrashReporting = enableCrashReporting
        self.enableLifecycleEvents = enableLifecycleEvents
        self.enableNetworkMonitoring = enableNetworkMonitoring
        self.retryBaseDelay = retryBaseDelay
        self.retryMaxDelay = retryMaxDelay
        self.retryMultiplier = retryMultiplier
        self.circuitBreakerThreshold = circuitBreakerThreshold
        self.circuitBreakerTimeout = circuitBreakerTimeout
    }
    
    /// Default configuration for development
    public static func development(
        serverURL: URL = URL(string: "http://localhost:8080")!,
        serviceName: String,
        agentID: String
    ) throws -> LoggerConfig {
        return try LoggerConfig(
            serverURL: serverURL,
            serviceName: serviceName,
            agentID: agentID,
            minimumLogLevel: .debug,
            flushInterval: 10.0
        )
    }
    
    /// Default configuration for production
    public static func production(
        serverURL: URL,
        serviceName: String,
        agentID: String
    ) throws -> LoggerConfig {
        return try LoggerConfig(
            serverURL: serverURL,
            serviceName: serviceName,
            agentID: agentID,
            minimumLogLevel: .info,
            flushInterval: 30.0,
            enableCrashReporting: true
        )
    }
}

/// Configuration errors
public enum LoggerConfigError: Error, LocalizedError {
    case invalidServiceName
    case invalidAgentID
    case invalidBufferSize
    case invalidFlushInterval
    case invalidTimeout
    
    public var errorDescription: String? {
        switch self {
        case .invalidServiceName:
            return "Service name cannot be empty"
        case .invalidAgentID:
            return "Agent ID cannot be empty"
        case .invalidBufferSize:
            return "Buffer size must be greater than 0"
        case .invalidFlushInterval:
            return "Flush interval must be greater than 0"
        case .invalidTimeout:
            return "Timeout must be greater than 0"
        }
    }
}