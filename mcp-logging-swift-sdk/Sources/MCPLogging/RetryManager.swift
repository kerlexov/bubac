import Foundation

/// Manages retry logic with exponential backoff
public class RetryManager {
    private let config: LoggerConfig
    private var retryCount = 0
    
    public init(config: LoggerConfig) {
        self.config = config
    }
    
    /// Executes an operation with retry logic
    /// - Parameter operation: The async operation to retry
    /// - Returns: The result of the operation
    public func retry<T>(_ operation: @escaping () async -> Result<T, HTTPClientError>) async -> Result<T, HTTPClientError> {
        retryCount = 0
        var delay = config.retryBaseDelay
        
        for attempt in 0..<config.maxRetries {
            let result = await operation()
            
            switch result {
            case .success(let value):
                retryCount = 0
                return .success(value)
                
            case .failure(let error):
                retryCount += 1
                
                // Don't retry on last attempt
                if attempt == config.maxRetries - 1 {
                    return .failure(error)
                }
                
                // Don't retry non-retryable errors
                guard error.isRetryable else {
                    return .failure(error)
                }
                
                // Calculate delay with jitter
                let jitteredDelay = addJitter(to: delay)
                let cappedDelay = min(jitteredDelay, config.retryMaxDelay)
                
                // Wait before retrying
                do {
                    try await Task.sleep(nanoseconds: UInt64(cappedDelay * 1_000_000_000))
                } catch {
                    return .failure(.networkError(URLError(.cancelled)))
                }
                
                // Exponential backoff
                delay *= config.retryMultiplier
            }
        }
        
        // This should never be reached, but just in case
        return .failure(.networkError(URLError(.unknown)))
    }
    
    /// Adds jitter to delay to prevent thundering herd
    private func addJitter(to delay: TimeInterval) -> TimeInterval {
        let jitter = Double.random(in: 0.0...0.1) * delay
        return delay + jitter
    }
    
    /// Resets the retry count
    public func reset() {
        retryCount = 0
    }
    
    /// Current retry count
    public var currentRetryCount: Int {
        return retryCount
    }
}

/// Circuit breaker implementation
public class CircuitBreaker {
    public enum State {
        case closed
        case open
        case halfOpen
    }
    
    private let config: LoggerConfig
    private var state: State = .closed
    private var failureCount = 0
    private var successCount = 0
    private var lastFailureTime: Date?
    private var lastStateChange: Date = Date()
    private let queue = DispatchQueue(label: "com.mcplogging.circuitbreaker")
    
    public init(config: LoggerConfig) {
        self.config = config
    }
    
    /// Executes an operation through the circuit breaker
    /// - Parameter operation: The operation to execute
    /// - Returns: The result of the operation or circuit breaker error
    public func execute<T>(_ operation: @escaping () async -> Result<T, HTTPClientError>) async -> Result<T, HTTPClientError> {
        // Check if we should attempt the operation
        let shouldAttempt = await queue.sync { () -> Bool in
            switch state {
            case .closed:
                return true
                
            case .open:
                // Check if we should transition to half-open
                if let lastFailure = lastFailureTime,
                   Date().timeIntervalSince(lastFailure) >= config.circuitBreakerTimeout {
                    state = .halfOpen
                    successCount = 0
                    lastStateChange = Date()
                    return true
                }
                return false
                
            case .halfOpen:
                return true
            }
        }
        
        guard shouldAttempt else {
            return .failure(.serverError(503, "Circuit breaker is open"))
        }
        
        // Execute the operation
        let result = await operation()
        
        // Update circuit breaker state based on result
        await queue.sync(flags: .barrier) {
            switch result {
            case .success:
                recordSuccess()
            case .failure(let error):
                if error.isRetryable {
                    recordFailure()
                }
            }
        }
        
        return result
    }
    
    private func recordSuccess() {
        successCount += 1
        
        switch state {
        case .closed:
            failureCount = 0
            
        case .halfOpen:
            // If we've had enough successes in half-open state, close the circuit
            if successCount >= 3 {
                state = .closed
                failureCount = 0
                successCount = 0
                lastStateChange = Date()
            }
            
        case .open:
            // Should not happen, but reset if it does
            state = .closed
            failureCount = 0
            successCount = 0
            lastStateChange = Date()
        }
    }
    
    private func recordFailure() {
        failureCount += 1
        lastFailureTime = Date()
        
        switch state {
        case .closed:
            if failureCount >= config.circuitBreakerThreshold {
                state = .open
                lastStateChange = Date()
            }
            
        case .halfOpen:
            // Immediately open on failure in half-open state
            state = .open
            lastStateChange = Date()
            
        case .open:
            // Already open, just update failure time
            break
        }
    }
    
    /// Current state of the circuit breaker
    public var currentState: State {
        return queue.sync {
            return state
        }
    }
    
    /// Current failure count
    public var currentFailureCount: Int {
        return queue.sync {
            return failureCount
        }
    }
    
    /// Forces the circuit breaker to open state
    public func forceOpen() {
        queue.sync(flags: .barrier) {
            state = .open
            lastFailureTime = Date()
            lastStateChange = Date()
        }
    }
    
    /// Forces the circuit breaker to closed state
    public func forceClose() {
        queue.sync(flags: .barrier) {
            state = .closed
            failureCount = 0
            successCount = 0
            lastFailureTime = nil
            lastStateChange = Date()
        }
    }
}