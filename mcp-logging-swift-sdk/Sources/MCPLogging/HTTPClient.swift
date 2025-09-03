import Foundation

/// HTTP client for communicating with the MCP logging server
public class HTTPClient {
    private let serverURL: URL
    private let session: URLSession
    private let timeout: TimeInterval
    private let encoder: JSONEncoder
    private let decoder: JSONDecoder
    
    public init(config: LoggerConfig) {
        self.serverURL = config.serverURL
        self.timeout = config.timeout
        
        // Configure URLSession
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = timeout
        configuration.timeoutIntervalForResource = timeout * 2
        configuration.requestCachePolicy = .reloadIgnoringLocalCacheData
        
        self.session = URLSession(configuration: configuration)
        
        // Configure JSON encoder/decoder
        self.encoder = JSONEncoder()
        self.decoder = JSONDecoder()
        
        // Configure date formatting
        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd'T'HH:mm:ss.SSSZ"
        dateFormatter.timeZone = TimeZone(secondsFromGMT: 0)
        
        encoder.dateEncodingStrategy = .formatted(dateFormatter)
        decoder.dateDecodingStrategy = .formatted(dateFormatter)
    }
    
    /// Sends a batch of log entries to the server
    /// - Parameter entries: Array of log entries to send
    /// - Returns: Result indicating success or failure
    public func sendLogs(_ entries: [LogEntry]) async -> Result<Void, HTTPClientError> {
        guard !entries.isEmpty else {
            return .success(())
        }
        
        let batch = LogBatch(logs: entries)
        
        do {
            // Encode the log batch
            let data = try encoder.encode(batch)
            
            // Create the request
            var request = URLRequest(url: serverURL.appendingPathComponent("/api/logs"))
            request.httpMethod = "POST"
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.setValue("mcp-logging-swift-sdk/1.0.0", forHTTPHeaderField: "User-Agent")
            request.httpBody = data
            
            // Send the request
            let (responseData, response) = try await session.data(for: request)
            
            // Check response status
            guard let httpResponse = response as? HTTPURLResponse else {
                return .failure(.invalidResponse)
            }
            
            switch httpResponse.statusCode {
            case 200...299:
                return .success(())
            case 400...499:
                let errorMessage = String(data: responseData, encoding: .utf8) ?? "Client error"
                return .failure(.clientError(httpResponse.statusCode, errorMessage))
            case 500...599:
                let errorMessage = String(data: responseData, encoding: .utf8) ?? "Server error"
                return .failure(.serverError(httpResponse.statusCode, errorMessage))
            default:
                return .failure(.unexpectedStatusCode(httpResponse.statusCode))
            }
            
        } catch let error as URLError {
            return .failure(.networkError(error))
        } catch {
            return .failure(.encodingError(error))
        }
    }
    
    /// Performs a health check against the server
    /// - Returns: Result indicating if the server is healthy
    public func healthCheck() async -> Result<HealthResponse, HTTPClientError> {
        do {
            let healthURL = serverURL.appendingPathComponent("/health")
            var request = URLRequest(url: healthURL)
            request.httpMethod = "GET"
            request.setValue("mcp-logging-swift-sdk/1.0.0", forHTTPHeaderField: "User-Agent")
            
            let (data, response) = try await session.data(for: request)
            
            guard let httpResponse = response as? HTTPURLResponse else {
                return .failure(.invalidResponse)
            }
            
            switch httpResponse.statusCode {
            case 200:
                do {
                    let healthResponse = try decoder.decode(HealthResponse.self, from: data)
                    return .success(healthResponse)
                } catch {
                    // Fallback to simple healthy response if JSON parsing fails
                    return .success(HealthResponse(status: "healthy", timestamp: Date()))
                }
            default:
                return .failure(.serverError(httpResponse.statusCode, "Health check failed"))
            }
            
        } catch let error as URLError {
            return .failure(.networkError(error))
        } catch {
            return .failure(.encodingError(error))
        }
    }
}

/// HTTP client errors
public enum HTTPClientError: Error, LocalizedError {
    case networkError(URLError)
    case encodingError(Error)
    case invalidResponse
    case clientError(Int, String)
    case serverError(Int, String)
    case unexpectedStatusCode(Int)
    
    public var errorDescription: String? {
        switch self {
        case .networkError(let urlError):
            return "Network error: \(urlError.localizedDescription)"
        case .encodingError(let error):
            return "Encoding error: \(error.localizedDescription)"
        case .invalidResponse:
            return "Invalid response from server"
        case .clientError(let code, let message):
            return "Client error \(code): \(message)"
        case .serverError(let code, let message):
            return "Server error \(code): \(message)"
        case .unexpectedStatusCode(let code):
            return "Unexpected status code: \(code)"
        }
    }
    
    /// Returns true if this is a retryable error
    public var isRetryable: Bool {
        switch self {
        case .networkError(let urlError):
            // Retry on timeout, connection lost, network unavailable
            return [
                URLError.timedOut,
                URLError.networkConnectionLost,
                URLError.notConnectedToInternet,
                URLError.dnsLookupFailed,
                URLError.cannotFindHost,
                URLError.cannotConnectToHost
            ].contains(urlError.code)
        case .serverError(let code, _):
            // Retry on 5xx server errors
            return code >= 500
        case .encodingError, .invalidResponse, .clientError, .unexpectedStatusCode:
            return false
        }
    }
}

/// Health response from the server
public struct HealthResponse: Codable {
    public let status: String
    public let timestamp: Date
    public let details: [String: String]?
    
    public init(status: String, timestamp: Date, details: [String: String]? = nil) {
        self.status = status
        self.timestamp = timestamp
        self.details = details
    }
    
    public var isHealthy: Bool {
        return status.lowercased() == "healthy"
    }
}