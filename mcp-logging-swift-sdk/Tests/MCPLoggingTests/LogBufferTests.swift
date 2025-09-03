import XCTest
@testable import MCPLogging

final class LogBufferTests: XCTestCase {
    
    var buffer: LogBuffer!
    
    override func setUp() {
        super.setUp()
        buffer = LogBuffer(maxSize: 3)
    }
    
    override func tearDown() {
        buffer = nil
        super.tearDown()
    }
    
    func testBufferInitialization() {
        XCTAssertEqual(buffer.count, 0)
        XCTAssertTrue(buffer.isEmpty)
        XCTAssertFalse(buffer.isFull)
    }
    
    func testAddingSingleEntry() {
        let entry = createTestLogEntry(message: "test message")
        
        let success = buffer.add(entry)
        
        XCTAssertTrue(success)
        XCTAssertEqual(buffer.count, 1)
        XCTAssertFalse(buffer.isEmpty)
        XCTAssertFalse(buffer.isFull)
    }
    
    func testBufferRotation() {
        // Fill buffer to capacity
        for i in 1...3 {
            let entry = createTestLogEntry(message: "message \(i)")
            buffer.add(entry)
        }
        
        XCTAssertTrue(buffer.isFull)
        XCTAssertEqual(buffer.count, 3)
        
        // Add one more entry, should rotate
        let newEntry = createTestLogEntry(message: "message 4")
        buffer.add(newEntry)
        
        XCTAssertEqual(buffer.count, 3) // Still at capacity
        
        let entries = buffer.peek()
        XCTAssertEqual(entries.first?.message, "message 2") // First entry rotated out
        XCTAssertEqual(entries.last?.message, "message 4") // New entry added
    }
    
    func testFlushAllEntries() {
        // Add some entries
        for i in 1...3 {
            let entry = createTestLogEntry(message: "message \(i)")
            buffer.add(entry)
        }
        
        let flushed = buffer.flush()
        
        XCTAssertEqual(flushed.count, 3)
        XCTAssertEqual(buffer.count, 0)
        XCTAssertTrue(buffer.isEmpty)
        
        // Check that entries are in correct order
        XCTAssertEqual(flushed[0].message, "message 1")
        XCTAssertEqual(flushed[1].message, "message 2")
        XCTAssertEqual(flushed[2].message, "message 3")
    }
    
    func testFlushCountLimited() {
        // Add some entries
        for i in 1...3 {
            let entry = createTestLogEntry(message: "message \(i)")
            buffer.add(entry)
        }
        
        let flushed = buffer.flush(count: 2)
        
        XCTAssertEqual(flushed.count, 2)
        XCTAssertEqual(buffer.count, 1)
        
        // Check that correct entries were flushed
        XCTAssertEqual(flushed[0].message, "message 1")
        XCTAssertEqual(flushed[1].message, "message 2")
        
        // Check remaining entry
        let remaining = buffer.peek()
        XCTAssertEqual(remaining.first?.message, "message 3")
    }
    
    func testPeekOperations() {
        let entry1 = createTestLogEntry(message: "first")
        let entry2 = createTestLogEntry(message: "second")
        
        buffer.add(entry1)
        buffer.add(entry2)
        
        XCTAssertEqual(buffer.peekOldest()?.message, "first")
        XCTAssertEqual(buffer.peekNewest()?.message, "second")
        
        // Peek should not modify buffer
        XCTAssertEqual(buffer.count, 2)
    }
    
    func testClear() {
        // Add some entries
        for i in 1...3 {
            let entry = createTestLogEntry(message: "message \(i)")
            buffer.add(entry)
        }
        
        buffer.clear()
        
        XCTAssertEqual(buffer.count, 0)
        XCTAssertTrue(buffer.isEmpty)
    }
    
    func testAddBatch() {
        let entries = [
            createTestLogEntry(message: "batch 1"),
            createTestLogEntry(message: "batch 2"),
            createTestLogEntry(message: "batch 3")
        ]
        
        let added = buffer.addBatch(entries)
        
        XCTAssertEqual(added, 3)
        XCTAssertEqual(buffer.count, 3)
        XCTAssertTrue(buffer.isFull)
    }
    
    // MARK: - Helper Methods
    
    private func createTestLogEntry(message: String) -> LogEntry {
        return LogEntry(
            level: .info,
            message: message,
            serviceName: "test-service",
            agentID: "test-agent",
            platform: "iOS"
        )
    }
}