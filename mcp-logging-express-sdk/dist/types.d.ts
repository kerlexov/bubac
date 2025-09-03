export interface LogEntry {
    id: string;
    timestamp: Date;
    level: LogLevel;
    message: string;
    serviceName: string;
    agentId: string;
    platform: string;
    metadata?: Record<string, any>;
    deviceInfo?: DeviceInfo;
    stackTrace?: string;
    sourceLocation?: SourceLocation;
}
export interface DeviceInfo {
    platform: string;
    version: string;
    model?: string;
    appVersion?: string;
}
export interface SourceLocation {
    file: string;
    line?: number;
    function?: string;
}
export declare enum LogLevel {
    DEBUG = "DEBUG",
    INFO = "INFO",
    WARN = "WARN",
    ERROR = "ERROR",
    FATAL = "FATAL"
}
export interface MCPLoggerConfig {
    serverUrl: string;
    serviceName: string;
    agentId: string;
    bufferSize?: number;
    flushInterval?: number;
    retryConfig?: RetryConfig;
    enableHealthCheck?: boolean;
    healthCheckPort?: number;
}
export interface RetryConfig {
    maxRetries: number;
    initialDelay: number;
    maxDelay: number;
    backoffMultiplier: number;
}
export interface MiddlewareOptions extends MCPLoggerConfig {
    logRequests?: boolean;
    logResponses?: boolean;
    includeHeaders?: boolean;
    includeBody?: boolean;
    excludePaths?: string[];
    sensitiveHeaders?: string[];
}
export interface LoggerAdapter {
    debug(message: string, metadata?: Record<string, any>): void;
    info(message: string, metadata?: Record<string, any>): void;
    warn(message: string, metadata?: Record<string, any>): void;
    error(message: string, metadata?: Record<string, any>): void;
    fatal(message: string, metadata?: Record<string, any>): void;
}
//# sourceMappingURL=types.d.ts.map