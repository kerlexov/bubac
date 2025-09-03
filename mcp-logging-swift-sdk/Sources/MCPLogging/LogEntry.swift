import Foundation

/// Device information for log entries
public struct DeviceInfo: Codable, Equatable {
    public let platform: String
    public let version: String
    public let model: String
    public let appVersion: String
    public let buildNumber: String?
    public let deviceName: String?
    public let systemName: String
    
    public init(
        platform: String,
        version: String,
        model: String,
        appVersion: String,
        buildNumber: String? = nil,
        deviceName: String? = nil,
        systemName: String
    ) {
        self.platform = platform
        self.version = version
        self.model = model
        self.appVersion = appVersion
        self.buildNumber = buildNumber
        self.deviceName = deviceName
        self.systemName = systemName
    }
    
    enum CodingKeys: String, CodingKey {
        case platform
        case version
        case model
        case appVersion = "app_version"
        case buildNumber = "build_number"
        case deviceName = "device_name"
        case systemName = "system_name"
    }
}

/// Source location information for log entries
public struct SourceLocation: Codable, Equatable {
    public let file: String
    public let line: Int
    public let function: String
    
    public init(file: String, line: Int, function: String) {
        self.file = file
        self.line = line
        self.function = function
    }
}

/// Log entry structure matching the MCP logging server format
public struct LogEntry: Codable, Equatable {
    public let id: String
    public let timestamp: Date
    public let level: LogLevel
    public let message: String
    public let serviceName: String
    public let agentID: String
    public let platform: String
    public let metadata: [String: AnyCodable]?
    public let deviceInfo: DeviceInfo?
    public let stackTrace: String?
    public let sourceLocation: SourceLocation?
    
    public init(
        id: String = UUID().uuidString,
        timestamp: Date = Date(),
        level: LogLevel,
        message: String,
        serviceName: String,
        agentID: String,
        platform: String,
        metadata: [String: Any]? = nil,
        deviceInfo: DeviceInfo? = nil,
        stackTrace: String? = nil,
        sourceLocation: SourceLocation? = nil
    ) {
        self.id = id
        self.timestamp = timestamp
        self.level = level
        self.message = message
        self.serviceName = serviceName
        self.agentID = agentID
        self.platform = platform
        self.metadata = metadata?.mapValues { AnyCodable($0) }
        self.deviceInfo = deviceInfo
        self.stackTrace = stackTrace
        self.sourceLocation = sourceLocation
    }
    
    enum CodingKeys: String, CodingKey {
        case id
        case timestamp
        case level
        case message
        case serviceName = "service_name"
        case agentID = "agent_id"
        case platform
        case metadata
        case deviceInfo = "device_info"
        case stackTrace = "stack_trace"
        case sourceLocation = "source_location"
    }
}

/// Type-erased codable wrapper for metadata values
public struct AnyCodable: Codable, Equatable {
    public let value: Any
    
    public init(_ value: Any) {
        self.value = value
    }
    
    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        
        if let boolValue = try? container.decode(Bool.self) {
            value = boolValue
        } else if let intValue = try? container.decode(Int.self) {
            value = intValue
        } else if let doubleValue = try? container.decode(Double.self) {
            value = doubleValue
        } else if let stringValue = try? container.decode(String.self) {
            value = stringValue
        } else if let arrayValue = try? container.decode([AnyCodable].self) {
            value = arrayValue.map { $0.value }
        } else if let dictionaryValue = try? container.decode([String: AnyCodable].self) {
            value = dictionaryValue.mapValues { $0.value }
        } else if container.decodeNil() {
            value = NSNull()
        } else {
            throw DecodingError.dataCorrupted(
                DecodingError.Context(
                    codingPath: decoder.codingPath,
                    debugDescription: "Unable to decode value"
                )
            )
        }
    }
    
    public func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        
        switch value {
        case let boolValue as Bool:
            try container.encode(boolValue)
        case let intValue as Int:
            try container.encode(intValue)
        case let doubleValue as Double:
            try container.encode(doubleValue)
        case let floatValue as Float:
            try container.encode(floatValue)
        case let stringValue as String:
            try container.encode(stringValue)
        case let arrayValue as [Any]:
            try container.encode(arrayValue.map { AnyCodable($0) })
        case let dictionaryValue as [String: Any]:
            try container.encode(dictionaryValue.mapValues { AnyCodable($0) })
        case is NSNull:
            try container.encodeNil()
        default:
            // Fallback to string representation
            try container.encode(String(describing: value))
        }
    }
    
    public static func == (lhs: AnyCodable, rhs: AnyCodable) -> Bool {
        switch (lhs.value, rhs.value) {
        case let (left as Bool, right as Bool):
            return left == right
        case let (left as Int, right as Int):
            return left == right
        case let (left as Double, right as Double):
            return left == right
        case let (left as String, right as String):
            return left == right
        case (is NSNull, is NSNull):
            return true
        default:
            return false
        }
    }
}

/// Batch of log entries for transmission
public struct LogBatch: Codable {
    public let logs: [LogEntry]
    
    public init(logs: [LogEntry]) {
        self.logs = logs
    }
}