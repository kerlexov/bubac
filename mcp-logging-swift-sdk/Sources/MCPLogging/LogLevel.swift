import Foundation

/// Log levels for MCP logging system
public enum LogLevel: String, CaseIterable, Codable {
    case debug = "DEBUG"
    case info = "INFO"
    case warn = "WARN"
    case error = "ERROR"
    case fatal = "FATAL"
    
    /// Priority order for log levels
    public var priority: Int {
        switch self {
        case .debug: return 0
        case .info: return 1
        case .warn: return 2
        case .error: return 3
        case .fatal: return 4
        }
    }
    
    /// Returns true if this level should be logged based on minimum level
    public func shouldLog(minimumLevel: LogLevel) -> Bool {
        return self.priority >= minimumLevel.priority
    }
}