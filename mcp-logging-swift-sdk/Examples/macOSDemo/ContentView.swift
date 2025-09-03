import SwiftUI
import MCPLogging

struct ContentView: View {
    @State private var logEntries: [String] = []
    @State private var isLoggingStarted = false
    
    private var logger: MCPLogger? {
        MCPLogging.shared
    }
    
    var body: some View {
        VStack {
            Text("MCP Logging macOS Demo")
                .font(.largeTitle)
                .padding()
            
            HStack {
                Button(isLoggingStarted ? "Stop Logging" : "Start Logging") {
                    if isLoggingStarted {
                        stopLogging()
                    } else {
                        startLogging()
                    }
                }
                .buttonStyle(.borderedProminent)
                
                Button("Generate Sample Logs") {
                    generateSampleLogs()
                }
                .buttonStyle(.bordered)
                .disabled(!isLoggingStarted)
                
                Button("Flush Logs") {
                    Task {
                        await logger?.flush()
                    }
                }
                .buttonStyle(.bordered)
                .disabled(!isLoggingStarted)
            }
            .padding()
            
            Text("Recent Log Entries:")
                .font(.headline)
                .padding(.top)
            
            ScrollView {
                VStack(alignment: .leading, spacing: 4) {
                    ForEach(logEntries.reversed(), id: \.self) { entry in
                        Text(entry)
                            .font(.system(.body, design: .monospaced))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2)
                            .background(Color.gray.opacity(0.1))
                            .cornerRadius(4)
                    }
                }
                .padding()
            }
            .frame(maxHeight: .infinity)
        }
        .frame(minWidth: 600, minHeight: 400)
        .padding()
    }
    
    private func startLogging() {
        do {
            let config = try LoggerConfig.development(
                serviceName: "macos-demo-app",
                agentID: "macos-demo-\(ProcessInfo.processInfo.processIdentifier)"
            )
            
            let logger = try MCPLogging.configure(with: config)
            logger.start()
            
            isLoggingStarted = true
            
            addLogEntry("Logger started with config: \(config.serviceName)")
            
            logger.info("macOS Demo app started", metadata: [
                "platform": "macOS",
                "process_id": ProcessInfo.processInfo.processIdentifier,
                "system_version": ProcessInfo.processInfo.operatingSystemVersionString
            ])
            
        } catch {
            addLogEntry("Failed to start logger: \(error)")
        }
    }
    
    private func stopLogging() {
        Task {
            await logger?.stop()
            isLoggingStarted = false
            addLogEntry("Logger stopped")
        }
    }
    
    private func generateSampleLogs() {
        guard let logger = logger else { return }
        
        // Generate different types of log entries
        logger.debug("Debug: Application state updated", metadata: [
            "component": "ui",
            "action": "state_change"
        ])
        
        logger.info("Info: User performed action", metadata: [
            "user_id": "demo_user",
            "action": "button_click",
            "timestamp": Date().timeIntervalSince1970
        ])
        
        logger.warn("Warn: Network latency detected", metadata: [
            "latency_ms": 250,
            "endpoint": "/api/logs",
            "retry_count": 1
        ])
        
        logger.error("Error: Failed to process request", metadata: [
            "error_code": "E001",
            "request_id": UUID().uuidString,
            "details": "Connection timeout"
        ])
        
        // Simulate system information logging
        logger.info("System information", metadata: [
            "memory_usage": ProcessInfo.processInfo.physicalMemory / 1024 / 1024,
            "active_processor_count": ProcessInfo.processInfo.activeProcessorCount,
            "thermal_state": ProcessInfo.processInfo.thermalState.rawValue
        ])
        
        addLogEntry("Generated \(5) sample log entries")
    }
    
    private func addLogEntry(_ entry: String) {
        let timestamp = DateFormatter.localizedString(
            from: Date(),
            dateStyle: .none,
            timeStyle: .medium
        )
        
        logEntries.append("[\(timestamp)] \(entry)")
        
        // Keep only last 50 entries
        if logEntries.count > 50 {
            logEntries.removeFirst(logEntries.count - 50)
        }
    }
}