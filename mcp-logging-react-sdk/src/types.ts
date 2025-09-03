export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR' | 'FATAL';

export interface LogEntry {
  id?: string;
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
  model: string;
  appVersion: string;
  userAgent?: string;
  screenResolution?: string;
  language?: string;
  timezone?: string;
}

export interface SourceLocation {
  file: string;
  line: number;
  function: string;
}

export interface MCPLoggerConfig {
  serverUrl: string;
  serviceName: string;
  agentId: string;
  bufferSize?: number;
  flushInterval?: number;
  retryAttempts?: number;
  retryDelay?: number;
  enableConsoleCapture?: boolean;
  enableErrorBoundary?: boolean;
  enablePerformanceMetrics?: boolean;
  enableUserInteractions?: boolean;
  enableLocalStorage?: boolean;
  logLevel?: LogLevel;
}

export interface PerformanceMetrics {
  pageLoadTime?: number;
  domContentLoadedTime?: number;
  firstContentfulPaint?: number;
  largestContentfulPaint?: number;
  cumulativeLayoutShift?: number;
  firstInputDelay?: number;
}

export interface UserInteraction {
  type: 'click' | 'navigation' | 'scroll' | 'input' | 'custom';
  element?: string;
  url?: string;
  timestamp: Date;
  metadata?: Record<string, any>;
}

export interface StorageStats {
  entryCount: number;
  sizeBytes: number;
  maxEntries: number;
  maxSizeBytes: number;
  usagePercent: number;
}

export interface MCPLogger {
  debug(message: string, metadata?: Record<string, any>): void;
  info(message: string, metadata?: Record<string, any>): void;
  warn(message: string, metadata?: Record<string, any>): void;
  error(message: string, metadata?: Record<string, any>): void;
  fatal(message: string, metadata?: Record<string, any>): void;
  logPerformance(metrics: PerformanceMetrics): void;
  logUserInteraction(interaction: UserInteraction): void;
  flush(): Promise<void>;
  getHealthStatus(): { isHealthy: boolean; lastError?: string };
  getStorageStats(): StorageStats | null;
}