import Foundation
import os.log

#if os(iOS)
import UIKit
#elseif os(macOS)
import AppKit
#endif

/// Integrates with Apple's OSLog system for native logging
@available(iOS 10.0, macOS 10.12, watchOS 3.0, tvOS 10.0, *)
public class SystemLogger {
    private let osLog: OSLog
    private let config: LoggerConfig
    private var mcpLogger: MCPLogger?
    
    public init(config: LoggerConfig) {
        self.config = config
        self.osLog = OSLog(subsystem: config.serviceName, category: "MCPLogging")
    }
    
    /// Sets the MCP logger to forward logs to
    public func setMCPLogger(_ logger: MCPLogger) {
        self.mcpLogger = logger
    }
    
    /// Logs an entry to the system log
    public func log(_ entry: LogEntry) {
        let osLogType = convertToOSLogType(entry.level)
        let message = formatMessage(entry)
        
        if #available(iOS 12.0, macOS 10.14, watchOS 5.0, tvOS 12.0, *) {
            os_log(.info, log: osLog, "%{public}@", message)
        } else {
            // Fallback for older versions
            NSLog("[%@] %@", entry.level.rawValue, message)
        }
    }
    
    private func convertToOSLogType(_ level: LogLevel) -> OSLogType {
        switch level {
        case .debug:
            return .debug
        case .info:
            return .info
        case .warn:
            return .default
        case .error:
            return .error
        case .fatal:
            return .fault
        }
    }
    
    private func formatMessage(_ entry: LogEntry) -> String {
        var components: [String] = [entry.message]
        
        if let metadata = entry.metadata, !metadata.isEmpty {
            let metadataString = metadata.map { "\($0.key)=\($0.value.value)" }.joined(separator: ", ")
            components.append("[\(metadataString)]")
        }
        
        if let sourceLocation = entry.sourceLocation {
            components.append("(\(sourceLocation.file):\(sourceLocation.line))")
        }
        
        return components.joined(separator: " ")
    }
}

/// Crash reporter for capturing and logging unhandled exceptions
public class CrashReporter {
    private var mcpLogger: MCPLogger?
    private var previousHandler: (@convention(c) (NSException) -> Void)?
    
    public init() {
        setupCrashHandling()
    }
    
    /// Sets the MCP logger to send crash reports to
    public func setMCPLogger(_ logger: MCPLogger) {
        self.mcpLogger = logger
    }
    
    private func setupCrashHandling() {
        // Store previous exception handler
        previousHandler = NSGetUncaughtExceptionHandler()
        
        // Set our exception handler
        NSSetUncaughtExceptionHandler { [weak self] exception in
            self?.handleException(exception)
            
            // Call previous handler if it exists
            if let previous = self?.previousHandler {
                previous(exception)
            }
        }
        
        // Setup signal handlers
        setupSignalHandlers()
    }
    
    private func setupSignalHandlers() {
        let signals = [SIGABRT, SIGILL, SIGSEGV, SIGFPE, SIGBUS]
        
        for signal in signals {
            let oldHandler = Darwin.signal(signal) { [weak self] signalNumber in
                self?.handleSignal(signalNumber)
                exit(signalNumber)
            }
            
            // Store old handler if needed
            _ = oldHandler
        }
    }
    
    private func handleException(_ exception: NSException) {
        guard let logger = mcpLogger else { return }
        
        let metadata: [String: Any] = [
            "crash_type": "exception",
            "exception_name": exception.name.rawValue,
            "exception_reason": exception.reason ?? "Unknown reason",
            "user_info": exception.userInfo ?? [:],
            "stack_symbols": exception.callStackSymbols,
            "return_addresses": exception.callStackReturnAddresses.map { $0.intValue }
        ]
        
        logger.fatal("Uncaught exception occurred", metadata: metadata)
        
        // Force flush to ensure the crash log is sent
        Task {
            await logger.flush()
        }
    }
    
    private func handleSignal(_ signal: Int32) {
        guard let logger = mcpLogger else { return }
        
        let signalName = signalName(for: signal)
        let stackTrace = Thread.callStackSymbols
        
        let metadata: [String: Any] = [
            "crash_type": "signal",
            "signal_number": signal,
            "signal_name": signalName,
            "stack_trace": stackTrace,
            "thread_info": threadInfo()
        ]
        
        logger.fatal("Application crashed with signal \(signalName)", metadata: metadata)
        
        // Force flush (synchronous to ensure it completes before crash)
        let semaphore = DispatchSemaphore(value: 0)
        Task {
            await logger.flush()
            semaphore.signal()
        }
        _ = semaphore.wait(timeout: .now() + 5) // Wait up to 5 seconds
    }
    
    private func signalName(for signal: Int32) -> String {
        switch signal {
        case SIGABRT: return "SIGABRT"
        case SIGILL: return "SIGILL"
        case SIGSEGV: return "SIGSEGV"
        case SIGFPE: return "SIGFPE"
        case SIGBUS: return "SIGBUS"
        case SIGPIPE: return "SIGPIPE"
        default: return "UNKNOWN(\(signal))"
        }
    }
    
    private func threadInfo() -> [String: Any] {
        let thread = Thread.current
        return [
            "is_main_thread": thread.isMainThread,
            "thread_name": thread.name ?? "unnamed",
            "thread_priority": thread.threadPriority,
            "quality_of_service": qualityOfServiceString(thread.qualityOfService)
        ]
    }
    
    private func qualityOfServiceString(_ qos: QualityOfService) -> String {
        switch qos {
        case .userInteractive: return "user_interactive"
        case .userInitiated: return "user_initiated"
        case .default: return "default"
        case .utility: return "utility"
        case .background: return "background"
        @unknown default: return "unknown"
        }
    }
    
    deinit {
        // Restore previous exception handler
        NSSetUncaughtExceptionHandler(previousHandler)
    }
}

/// Background task manager for handling log transmission in background
#if os(iOS)
public class BackgroundTaskManager {
    private var currentTaskID: UIBackgroundTaskIdentifier?
    private let mcpLogger: MCPLogger
    
    public init(mcpLogger: MCPLogger) {
        self.mcpLogger = mcpLogger
    }
    
    /// Executes a task in the background
    public func performBackgroundTask(_ task: @escaping () async -> Void) {
        // Start background task
        currentTaskID = UIApplication.shared.beginBackgroundTask { [weak self] in
            self?.endCurrentBackgroundTask()
        }
        
        Task {
            await task()
            endCurrentBackgroundTask()
        }
    }
    
    /// Flushes logs in background task
    public func backgroundFlush() {
        performBackgroundTask { [weak self] in
            await self?.mcpLogger.flush()
        }
    }
    
    private func endCurrentBackgroundTask() {
        guard let taskID = currentTaskID else { return }
        UIApplication.shared.endBackgroundTask(taskID)
        currentTaskID = nil
    }
    
    deinit {
        endCurrentBackgroundTask()
    }
}
#endif