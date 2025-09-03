import XCTest
@testable import MCPLogging

final class LoggerConfigTests: XCTestCase {
    
    func testValidConfiguration() throws {
        let serverURL = URL(string: "http://localhost:8080")!
        
        let config = try LoggerConfig(
            serverURL: serverURL,
            serviceName: "test-service",
            agentID: "test-agent"
        )
        
        XCTAssertEqual(config.serverURL, serverURL)
        XCTAssertEqual(config.serviceName, "test-service")
        XCTAssertEqual(config.agentID, "test-agent")
        XCTAssertEqual(config.bufferSize, 1000)
        XCTAssertEqual(config.flushInterval, 30.0)
    }
    
    func testInvalidServiceName() {
        let serverURL = URL(string: "http://localhost:8080")!
        
        XCTAssertThrowsError(try LoggerConfig(
            serverURL: serverURL,
            serviceName: "",
            agentID: "test-agent"
        )) { error in
            XCTAssertTrue(error is LoggerConfigError)
        }
    }
    
    func testInvalidAgentID() {
        let serverURL = URL(string: "http://localhost:8080")!
        
        XCTAssertThrowsError(try LoggerConfig(
            serverURL: serverURL,
            serviceName: "test-service",
            agentID: ""
        )) { error in
            XCTAssertTrue(error is LoggerConfigError)
        }
    }
    
    func testDevelopmentConfiguration() throws {
        let config = try LoggerConfig.development(
            serviceName: "test-service",
            agentID: "test-agent"
        )
        
        XCTAssertEqual(config.serviceName, "test-service")
        XCTAssertEqual(config.agentID, "test-agent")
        XCTAssertEqual(config.minimumLogLevel, .debug)
        XCTAssertEqual(config.flushInterval, 10.0)
    }
    
    func testProductionConfiguration() throws {
        let serverURL = URL(string: "https://prod-server.com")!
        let config = try LoggerConfig.production(
            serverURL: serverURL,
            serviceName: "prod-service",
            agentID: "prod-agent"
        )
        
        XCTAssertEqual(config.serverURL, serverURL)
        XCTAssertEqual(config.serviceName, "prod-service")
        XCTAssertEqual(config.agentID, "prod-agent")
        XCTAssertEqual(config.minimumLogLevel, .info)
        XCTAssertEqual(config.flushInterval, 30.0)
        XCTAssertTrue(config.enableCrashReporting)
    }
}