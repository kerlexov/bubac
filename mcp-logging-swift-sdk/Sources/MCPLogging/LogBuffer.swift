import Foundation

/// Thread-safe buffer for storing log entries before transmission
public class LogBuffer {
    private let maxSize: Int
    private var entries: [LogEntry] = []
    private let queue = DispatchQueue(label: "com.mcplogging.buffer", attributes: .concurrent)
    
    public init(maxSize: Int) {
        self.maxSize = maxSize
    }
    
    /// Adds a log entry to the buffer
    /// - Parameter entry: The log entry to add
    /// - Returns: True if added successfully, false if buffer is full and entry was dropped
    @discardableResult
    public func add(_ entry: LogEntry) -> Bool {
        return queue.sync(flags: .barrier) {
            guard entries.count < maxSize else {
                // Implement rotation strategy - remove oldest entry
                entries.removeFirst()
            }
            
            entries.append(entry)
            return true
        }
    }
    
    /// Adds multiple log entries to the buffer
    /// - Parameter entries: Array of log entries to add
    /// - Returns: Number of entries successfully added
    @discardableResult
    public func addBatch(_ logEntries: [LogEntry]) -> Int {
        return queue.sync(flags: .barrier) {
            var added = 0
            for entry in logEntries {
                if entries.count >= maxSize {
                    entries.removeFirst()
                }
                entries.append(entry)
                added += 1
            }
            return added
        }
    }
    
    /// Removes and returns all buffered entries
    /// - Returns: Array of all buffered log entries
    public func flush() -> [LogEntry] {
        return queue.sync(flags: .barrier) {
            let result = entries
            entries.removeAll()
            return result
        }
    }
    
    /// Removes and returns up to the specified number of entries
    /// - Parameter count: Maximum number of entries to return
    /// - Returns: Array of log entries (up to count)
    public func flush(count: Int) -> [LogEntry] {
        return queue.sync(flags: .barrier) {
            let actualCount = min(count, entries.count)
            guard actualCount > 0 else { return [] }
            
            let result = Array(entries.prefix(actualCount))
            entries.removeFirst(actualCount)
            return result
        }
    }
    
    /// Returns the current number of buffered entries
    public var count: Int {
        return queue.sync {
            return entries.count
        }
    }
    
    /// Returns true if the buffer is empty
    public var isEmpty: Bool {
        return queue.sync {
            return entries.isEmpty
        }
    }
    
    /// Returns true if the buffer is at capacity
    public var isFull: Bool {
        return queue.sync {
            return entries.count >= maxSize
        }
    }
    
    /// Clears all buffered entries
    public func clear() {
        queue.sync(flags: .barrier) {
            entries.removeAll()
        }
    }
    
    /// Returns a copy of all buffered entries without removing them
    public func peek() -> [LogEntry] {
        return queue.sync {
            return entries
        }
    }
    
    /// Returns the oldest entry without removing it
    public func peekOldest() -> LogEntry? {
        return queue.sync {
            return entries.first
        }
    }
    
    /// Returns the newest entry without removing it
    public func peekNewest() -> LogEntry? {
        return queue.sync {
            return entries.last
        }
    }
}