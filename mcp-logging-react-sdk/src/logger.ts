import { LogEntry, LogLevel, MCPLogger, MCPLoggerConfig, PerformanceMetrics, UserInteraction, DeviceInfo } from './types';
import { BrowserFeatures } from './browser-features';
import { LocalStorageManager } from './storage-manager';

export class MCPLoggerImpl implements MCPLogger {
  private config: Required<MCPLoggerConfig>;
  private buffer: LogEntry[] = [];
  private flushTimer?: NodeJS.Timeout;
  private isHealthy = true;
  private lastError?: string;
  private retryCount = 0;
  private browserFeatures?: BrowserFeatures;
  private storageManager?: LocalStorageManager;

  constructor(config: MCPLoggerConfig) {
    this.config = {
      bufferSize: 100,
      flushInterval: 5000,
      retryAttempts: 3,
      retryDelay: 1000,
      enableConsoleCapture: true,
      enableErrorBoundary: true,
      enablePerformanceMetrics: true,
      enableUserInteractions: true,
      enableLocalStorage: true,
      logLevel: 'INFO',
      ...config,
    };

    this.initializeStorageManager();
    this.startFlushTimer();
    this.setupConsoleCapture();
    this.setupErrorCapture();
    this.loadBufferedLogs();
    this.initializeBrowserFeatures();
  }

  debug(message: string, metadata?: Record<string, any>): void {
    this.log('DEBUG', message, metadata);
  }

  info(message: string, metadata?: Record<string, any>): void {
    this.log('INFO', message, metadata);
  }

  warn(message: string, metadata?: Record<string, any>): void {
    this.log('WARN', message, metadata);
  }

  error(message: string, metadata?: Record<string, any>): void {
    this.log('ERROR', message, metadata);
  }

  fatal(message: string, metadata?: Record<string, any>): void {
    this.log('FATAL', message, metadata);
  }

  logPerformance(metrics: PerformanceMetrics): void {
    this.log('INFO', 'Performance metrics captured', { 
      type: 'performance',
      metrics 
    });
  }

  logUserInteraction(interaction: UserInteraction): void {
    this.log('INFO', `User interaction: ${interaction.type}`, {
      type: 'user_interaction',
      interaction
    });
  }

  private log(level: LogLevel, message: string, metadata?: Record<string, any>): void {
    if (!this.shouldLog(level)) {
      return;
    }

    const entry: LogEntry = {
      id: this.generateId(),
      timestamp: new Date(),
      level,
      message,
      serviceName: this.config.serviceName,
      agentId: this.config.agentId,
      platform: 'react',
      metadata,
      deviceInfo: this.getDeviceInfo(),
      sourceLocation: this.getSourceLocation(),
    };

    this.buffer.push(entry);

    if (this.buffer.length >= this.config.bufferSize) {
      this.flush();
    }
  }

  private shouldLog(level: LogLevel): boolean {
    const levels: LogLevel[] = ['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'];
    const currentLevelIndex = levels.indexOf(this.config.logLevel);
    const logLevelIndex = levels.indexOf(level);
    return logLevelIndex >= currentLevelIndex;
  }

  async flush(): Promise<void> {
    if (this.buffer.length === 0) {
      return;
    }

    const logsToSend = [...this.buffer];
    this.buffer = [];

    try {
      await this.sendLogs(logsToSend);
      this.isHealthy = true;
      this.lastError = undefined;
      this.retryCount = 0;
    } catch (error) {
      this.handleSendError(error, logsToSend);
    }
  }

  private async sendLogs(logs: LogEntry[]): Promise<void> {
    const response = await fetch(`${this.config.serverUrl}/api/logs`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ logs }),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
  }

  private handleSendError(error: any, logs: LogEntry[]): void {
    this.isHealthy = false;
    this.lastError = error.message || 'Unknown error';

    if (this.config.enableLocalStorage && this.storageManager) {
      this.storageManager.saveLogs(logs);
    }

    if (this.retryCount < this.config.retryAttempts) {
      this.retryCount++;
      const delay = this.config.retryDelay * Math.pow(2, this.retryCount - 1);
      
      setTimeout(() => {
        this.sendLogs(logs).catch(() => {
          // Final retry failed, logs are saved to localStorage
        });
      }, delay);
    }
  }

  private loadBufferedLogs(): void {
    if (!this.config.enableLocalStorage || !this.storageManager) {
      return;
    }

    try {
      const bufferedLogs = this.storageManager.loadLogs();
      if (bufferedLogs.length > 0) {
        this.buffer.push(...bufferedLogs);
        this.storageManager.clearLogs();
        
        // Attempt to send buffered logs
        this.flush();
      }
    } catch (error) {
      console.warn('Failed to load buffered logs from localStorage:', error);
    }
  }

  private initializeStorageManager(): void {
    if (this.config.enableLocalStorage && typeof window !== 'undefined') {
      this.storageManager = new LocalStorageManager({
        keyPrefix: `mcp-logger-${this.config.serviceName}`,
        maxEntries: this.config.bufferSize * 10, // Allow more storage than buffer
        maxSizeBytes: 5 * 1024 * 1024, // 5MB limit
      });
    }
  }

  private startFlushTimer(): void {
    this.flushTimer = setInterval(() => {
      this.flush();
    }, this.config.flushInterval);
  }

  private setupConsoleCapture(): void {
    if (!this.config.enableConsoleCapture) {
      return;
    }

    const originalConsole = { ...console };

    console.log = (...args) => {
      originalConsole.log(...args);
      this.info('Console log', { args: args.map(String) });
    };

    console.warn = (...args) => {
      originalConsole.warn(...args);
      this.warn('Console warn', { args: args.map(String) });
    };

    console.error = (...args) => {
      originalConsole.error(...args);
      this.error('Console error', { args: args.map(String) });
    };
  }

  private setupErrorCapture(): void {
    if (!this.config.enableErrorBoundary) {
      return;
    }

    window.addEventListener('error', (event) => {
      this.error('Unhandled error', {
        message: event.message,
        filename: event.filename,
        lineno: event.lineno,
        colno: event.colno,
        stack: event.error?.stack,
      });
    });

    window.addEventListener('unhandledrejection', (event) => {
      this.error('Unhandled promise rejection', {
        reason: String(event.reason),
        stack: event.reason?.stack,
      });
    });
  }

  private getDeviceInfo(): DeviceInfo {
    return {
      platform: 'Web',
      version: navigator.userAgent,
      model: 'Browser',
      appVersion: '1.0.0',
      userAgent: navigator.userAgent,
      screenResolution: `${screen.width}x${screen.height}`,
      language: navigator.language,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    };
  }

  private getSourceLocation(): any {
    try {
      const stack = new Error().stack;
      if (stack) {
        const lines = stack.split('\n');
        // Skip the first few lines (this function and log function)
        const callerLine = lines[4] || lines[3] || lines[2];
        const match = callerLine.match(/at\s+(.+?)\s+\((.+?):(\d+):(\d+)\)/);
        if (match) {
          return {
            function: match[1],
            file: match[2],
            line: parseInt(match[3]),
          };
        }
      }
    } catch (error) {
      // Ignore errors in source location detection
    }
    return undefined;
  }

  private generateId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  getHealthStatus(): { isHealthy: boolean; lastError?: string } {
    return {
      isHealthy: this.isHealthy,
      lastError: this.lastError,
    };
  }

  getStorageStats(): any {
    if (!this.storageManager) {
      return null;
    }
    return this.storageManager.getStorageStats();
  }

  destroy(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
    }
    if (this.browserFeatures) {
      this.browserFeatures.destroy();
    }
    this.flush();
  }

  private initializeBrowserFeatures(): void {
    if (typeof window === 'undefined') {
      return;
    }

    this.browserFeatures = new BrowserFeatures(this);
    this.browserFeatures.initialize();
    
    if (this.config.enableUserInteractions) {
      this.browserFeatures.setupEnhancedUserInteractionTracking();
    }
  }
}