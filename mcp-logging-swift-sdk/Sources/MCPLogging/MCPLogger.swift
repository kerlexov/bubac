import Foundation

#if os(iOS)
import UIKit
#elseif os(macOS)
import AppKit
#endif

/// Main MCP Logger class providing centralized logging functionality
public class MCPLogger {
    private let config: LoggerConfig
    private let httpClient: HTTPClient
    private let buffer: LogBuffer
    private let retryManager: RetryManager
    private let circuitBreaker: CircuitBreaker
    private let deviceInfo: DeviceInfo
    
    private var flushTimer: Timer?
    private var networkMonitor: AnyObject?
    private let logQueue = DispatchQueue(label: "com.mcplogging.logger", qos: .utility)
    
    private var isStarted = false
    private var backgroundTaskID: UIBackgroundTaskIdentifier?
    
    /// Initializes the MCP Logger with the provided configuration
    /// - Parameter config: Logger configuration
    public init(config: LoggerConfig) throws {
        self.config = config
        self.httpClient = HTTPClient(config: config)
        self.buffer = LogBuffer(maxSize: config.bufferSize)
        self.retryManager = RetryManager(config: config)
        self.circuitBreaker = CircuitBreaker(config: config)
        self.deviceInfo = DeviceInfoCollector.collect()
        
        try validateConfiguration(config)
    }
    
    /// Starts the logger and begins periodic flushing
    public func start() {
        guard !isStarted else { return }
        
        isStarted = true
        
        // Start periodic flushing
        startFlushTimer()
        
        // Setup network monitoring if enabled
        if config.enableNetworkMonitoring {
            setupNetworkMonitoring()
        }
        
        // Setup lifecycle monitoring if enabled
        if config.enableLifecycleEvents {
            setupLifecycleMonitoring()
        }
        
        // Log startup
        info("MCP Logger started", metadata: [
            "service_name": config.serviceName,
            "agent_id": config.agentID,
            "buffer_size": config.bufferSize,
            "flush_interval": config.flushInterval
        ])
    }
    
    /// Stops the logger and flushes remaining logs
    public func stop() async {
        guard isStarted else { return }
        
        isStarted = false
        
        // Stop timer
        flushTimer?.invalidate()
        flushTimer = nil
        
        // Stop network monitoring
        stopNetworkMonitoring()
        
        // Final flush
        await flush()
        
        info("MCP Logger stopped")
    }
    
    // MARK: - Logging Methods
    
    /// Logs a debug message
    /// - Parameters:
    ///   - message: The log message
    ///   - metadata: Optional metadata dictionary
    ///   - file: Source file (automatically filled)
    ///   - function: Source function (automatically filled)
    ///   - line: Source line (automatically filled)
    public func debug(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        log(level: .debug, message: message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Logs an info message
    /// - Parameters:
    ///   - message: The log message
    ///   - metadata: Optional metadata dictionary
    ///   - file: Source file (automatically filled)
    ///   - function: Source function (automatically filled)
    ///   - line: Source line (automatically filled)
    public func info(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        log(level: .info, message: message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Logs a warning message
    /// - Parameters:
    ///   - message: The log message
    ///   - metadata: Optional metadata dictionary
    ///   - file: Source file (automatically filled)
    ///   - function: Source function (automatically filled)
    ///   - line: Source line (automatically filled)
    public func warn(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        log(level: .warn, message: message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Logs an error message
    /// - Parameters:
    ///   - message: The log message
    ///   - metadata: Optional metadata dictionary
    ///   - file: Source file (automatically filled)
    ///   - function: Source function (automatically filled)
    ///   - line: Source line (automatically filled)
    public func error(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        log(level: .error, message: message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Logs a fatal message
    /// - Parameters:
    ///   - message: The log message
    ///   - metadata: Optional metadata dictionary
    ///   - file: Source file (automatically filled)
    ///   - function: Source function (automatically filled)
    ///   - line: Source line (automatically filled)
    public func fatal(
        _ message: String,
        metadata: [String: Any]? = nil,
        file: String = #file,
        function: String = #function,
        line: Int = #line
    ) {
        log(level: .fatal, message: message, metadata: metadata, file: file, function: function, line: line)
    }
    
    /// Generic log method
    private func log(
        level: LogLevel,
        message: String,
        metadata: [String: Any]?,
        file: String,
        function: String,
        line: Int
    ) {
        // Check if this level should be logged
        guard level.shouldLog(minimumLevel: config.minimumLogLevel) else {
            return
        }
        
        logQueue.async { [weak self] in
            guard let self = self else { return }
            
            let sourceLocation = SourceLocation(
                file: URL(fileURLWithPath: file).lastPathComponent,
                line: line,
                function: function
            )
            
            let entry = LogEntry(
                level: level,
                message: message,
                serviceName: self.config.serviceName,
                agentID: self.config.agentID,
                platform: DeviceInfoCollector.currentPlatform,
                metadata: metadata,
                deviceInfo: self.deviceInfo,
                sourceLocation: sourceLocation
            )
            
            self.buffer.add(entry)
        }
    }
    
    // MARK: - Flushing
    
    /// Manually flushes buffered logs to the server
    public func flush() async {
        let entries = buffer.flush()
        guard !entries.isEmpty else { return }
        
        await sendLogs(entries)
    }
    
    private func sendLogs(_ entries: [LogEntry]) async {
        let operation = { [weak self] () async -> Result<Void, HTTPClientError> in
            guard let self = self else {
                return .failure(.networkError(URLError(.cancelled)))
            }
            return await self.httpClient.sendLogs(entries)
        }
        
        let result = await circuitBreaker.execute {
            await retryManager.retry(operation)
        }
        
        switch result {
        case .success:
            // Logs sent successfully
            break
        case .failure(let error):
            // Re-buffer failed logs (up to a limit to prevent infinite growth)
            if buffer.count < config.bufferSize / 2 {
                buffer.addBatch(entries)
            }
            
            // Log the error (but avoid infinite recursion)
            if entries.first?.level != .error {
                self.error("Failed to send logs to server", metadata: [
                    "error": error.localizedDescription,
                    "entries_count": entries.count
                ])
            }
        }
    }
    
    private func startFlushTimer() {
        flushTimer = Timer.scheduledTimer(withTimeInterval: config.flushInterval, repeats: true) { [weak self] _ in
            guard let self = self else { return }
            
            Task {
                await self.flush()
            }
        }
    }
    
    // MARK: - Network Monitoring
    
    private func setupNetworkMonitoring() {
        if #available(iOS 12.0, macOS 10.14, watchOS 5.0, tvOS 12.0, *) {
            let monitor = NetworkMonitor()
            
            monitor.statusUpdateHandler = { [weak self] status in
                guard let self = self else { return }
                
                switch status {
                case .satisfied:
                    // Network is available, try to flush any pending logs
                    Task {
                        await self.flush()
                    }
                case .unsatisfied, .requiresConnection:
                    // Network is not available
                    break
                }
            }
            
            monitor.startMonitoring()
            networkMonitor = monitor
        } else {
            // Fallback for older versions
            let monitor = LegacyNetworkMonitor()
            
            monitor.statusUpdateHandler = { [weak self] status in
                guard let self = self else { return }
                
                if status == .reachable {
                    Task {
                        await self.flush()
                    }
                }
            }
            
            monitor.startMonitoring()
            networkMonitor = monitor
        }
    }
    
    private func stopNetworkMonitoring() {
        if #available(iOS 12.0, macOS 10.14, watchOS 5.0, tvOS 12.0, *) {
            (networkMonitor as? NetworkMonitor)?.stopMonitoring()
        } else {
            (networkMonitor as? LegacyNetworkMonitor)?.stopMonitoring()
        }
        networkMonitor = nil
    }
    
    // MARK: - Lifecycle Monitoring
    
    private func setupLifecycleMonitoring() {
        #if os(iOS)
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(appDidBecomeActive),
            name: UIApplication.didBecomeActiveNotification,
            object: nil
        )
        
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(appWillResignActive),
            name: UIApplication.willResignActiveNotification,
            object: nil
        )
        
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(appDidEnterBackground),
            name: UIApplication.didEnterBackgroundNotification,
            object: nil
        )
        
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(appWillEnterForeground),
            name: UIApplication.willEnterForegroundNotification,
            object: nil
        )
        #endif
    }
    
    #if os(iOS)
    @objc private func appDidBecomeActive() {
        info("Application became active", metadata: [
            "event": "lifecycle",
            "state": "active"
        ])
    }
    
    @objc private func appWillResignActive() {
        info("Application will resign active", metadata: [
            "event": "lifecycle",
            "state": "inactive"
        ])
    }
    
    @objc private func appDidEnterBackground() {
        info("Application entered background", metadata: [
            "event": "lifecycle",
            "state": "background"
        ])
        
        // Start background task to complete log transmission
        backgroundTaskID = UIApplication.shared.beginBackgroundTask { [weak self] in
            self?.endBackgroundTask()
        }
        
        Task {
            await flush()
            endBackgroundTask()
        }
    }
    
    @objc private func appWillEnterForeground() {
        info("Application will enter foreground", metadata: [
            "event": "lifecycle",
            "state": "foreground"
        ])
        
        endBackgroundTask()
    }
    
    private func endBackgroundTask() {
        guard let taskID = backgroundTaskID else { return }
        UIApplication.shared.endBackgroundTask(taskID)
        backgroundTaskID = nil
    }
    #endif
    
    // MARK: - Utilities
    
    private func validateConfiguration(_ config: LoggerConfig) throws {
        // Additional validation can be added here
    }
    
    /// Returns current buffer statistics
    public var bufferStats: [String: Any] {
        return [
            "count": buffer.count,
            "is_full": buffer.isFull,
            "max_size": config.bufferSize
        ]
    }
    
    /// Returns current circuit breaker status
    public var circuitBreakerStatus: [String: Any] {
        return [
            "state": String(describing: circuitBreaker.currentState),
            "failure_count": circuitBreaker.currentFailureCount
        ]
    }
    
    /// Performs a health check against the server
    public func healthCheck() async -> Result<HealthResponse, HTTPClientError> {
        return await httpClient.healthCheck()
    }
    
    deinit {
        if isStarted {
            flushTimer?.invalidate()
            stopNetworkMonitoring()
            NotificationCenter.default.removeObserver(self)
        }
    }
}