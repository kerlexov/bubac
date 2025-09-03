import SwiftUI
import MCPLogging

struct ContentView: View {
    @State private var logMessage = "Hello, MCP Logging!"
    @State private var logLevel = LogLevel.info
    @State private var customMetadata = ""
    @State private var bufferStats: [String: Any] = [:]
    @State private var circuitBreakerStatus: [String: Any] = [:]
    
    private var logger: MCPLogger? {
        MCPLogging.shared
    }
    
    var body: some View {
        NavigationView {
            VStack(spacing: 20) {
                Text("MCP Logging Demo")
                    .font(.largeTitle)
                    .padding()
                
                VStack(alignment: .leading, spacing: 10) {
                    Text("Log Message:")
                        .font(.headline)
                    
                    TextField("Enter log message", text: $logMessage)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                    
                    Text("Log Level:")
                        .font(.headline)
                    
                    Picker("Log Level", selection: $logLevel) {
                        Text("Debug").tag(LogLevel.debug)
                        Text("Info").tag(LogLevel.info)
                        Text("Warn").tag(LogLevel.warn)
                        Text("Error").tag(LogLevel.error)
                        Text("Fatal").tag(LogLevel.fatal)
                    }
                    .pickerStyle(SegmentedPickerStyle())
                    
                    Text("Custom Metadata (JSON):")
                        .font(.headline)
                    
                    TextField("e.g., {\"user_id\": 123}", text: $customMetadata)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                }
                .padding()
                
                Button("Send Log") {
                    sendLog()
                }
                .buttonStyle(.borderedProminent)
                .padding()
                
                Group {
                    Button("Test Crash Reporting") {
                        testCrashReporting()
                    }
                    .buttonStyle(.bordered)
                    
                    Button("Flush Logs") {
                        Task {
                            await logger?.flush()
                            updateStats()
                        }
                    }
                    .buttonStyle(.bordered)
                    
                    Button("Health Check") {
                        Task {
                            await performHealthCheck()
                        }
                    }
                    .buttonStyle(.bordered)
                }
                
                VStack(alignment: .leading, spacing: 10) {
                    Text("Buffer Stats:")
                        .font(.headline)
                    
                    ForEach(bufferStats.keys.sorted(), id: \.self) { key in
                        HStack {
                            Text(key + ":")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Spacer()
                            Text(String(describing: bufferStats[key] ?? ""))
                                .font(.caption)
                                .foregroundColor(.primary)
                        }
                    }
                    
                    Text("Circuit Breaker:")
                        .font(.headline)
                        .padding(.top)
                    
                    ForEach(circuitBreakerStatus.keys.sorted(), id: \.self) { key in
                        HStack {
                            Text(key + ":")
                                .font(.caption)
                                .foregroundColor(.secondary)
                            Spacer()
                            Text(String(describing: circuitBreakerStatus[key] ?? ""))
                                .font(.caption)
                                .foregroundColor(.primary)
                        }
                    }
                }
                .padding()
                .background(Color.gray.opacity(0.1))
                .cornerRadius(8)
                
                Spacer()
            }
            .padding()
            .onAppear {
                setupLogger()
                updateStats()
            }
            .navigationTitle("MCP Logging")
            .navigationBarTitleDisplayMode(.inline)
        }
    }
    
    private func setupLogger() {
        do {
            let config = try LoggerConfig.development(
                serviceName: "ios-demo-app",
                agentID: "demo-\(UUID().uuidString.prefix(8))"
            )
            
            let logger = try MCPLogging.configure(with: config)
            logger.start()
            
            logger.info("Demo app started", metadata: [
                "platform": "iOS",
                "app_version": "1.0.0"
            ])
            
        } catch {
            print("Failed to setup logger: \(error)")
        }
    }
    
    private func sendLog() {
        guard let logger = logger else { return }
        
        var metadata: [String: Any]? = nil
        
        if !customMetadata.isEmpty {
            if let data = customMetadata.data(using: .utf8),
               let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                metadata = json
            }
        }
        
        switch logLevel {
        case .debug:
            logger.debug(logMessage, metadata: metadata)
        case .info:
            logger.info(logMessage, metadata: metadata)
        case .warn:
            logger.warn(logMessage, metadata: metadata)
        case .error:
            logger.error(logMessage, metadata: metadata)
        case .fatal:
            logger.fatal(logMessage, metadata: metadata)
        }
        
        updateStats()
    }
    
    private func testCrashReporting() {
        guard let logger = logger else { return }
        
        logger.error("Testing crash reporting", metadata: [
            "test": true,
            "timestamp": Date().timeIntervalSince1970
        ])
        
        // Simulate an error condition
        logger.fatal("Simulated crash condition", metadata: [
            "crash_type": "simulation",
            "stack_trace": Thread.callStackSymbols
        ])
    }
    
    private func performHealthCheck() async {
        guard let logger = logger else { return }
        
        let result = await logger.healthCheck()
        
        switch result {
        case .success(let response):
            logger.info("Health check successful", metadata: [
                "status": response.status,
                "timestamp": response.timestamp.timeIntervalSince1970
            ])
        case .failure(let error):
            logger.error("Health check failed", metadata: [
                "error": error.localizedDescription
            ])
        }
        
        updateStats()
    }
    
    private func updateStats() {
        guard let logger = logger else { return }
        
        bufferStats = logger.bufferStats
        circuitBreakerStatus = logger.circuitBreakerStatus
    }
}