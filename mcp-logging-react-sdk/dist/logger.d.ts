import { MCPLogger, MCPLoggerConfig, PerformanceMetrics, UserInteraction } from './types';
export declare class MCPLoggerImpl implements MCPLogger {
    private config;
    private buffer;
    private flushTimer?;
    private isHealthy;
    private lastError?;
    private retryCount;
    constructor(config: MCPLoggerConfig);
    debug(message: string, metadata?: Record<string, any>): void;
    info(message: string, metadata?: Record<string, any>): void;
    warn(message: string, metadata?: Record<string, any>): void;
    error(message: string, metadata?: Record<string, any>): void;
    fatal(message: string, metadata?: Record<string, any>): void;
    logPerformance(metrics: PerformanceMetrics): void;
    logUserInteraction(interaction: UserInteraction): void;
    private log;
    private shouldLog;
    flush(): Promise<void>;
    private sendLogs;
    private handleSendError;
    private saveToLocalStorage;
    private loadBufferedLogs;
    private startFlushTimer;
    private setupConsoleCapture;
    private setupErrorCapture;
    private getDeviceInfo;
    private getSourceLocation;
    private generateId;
    getHealthStatus(): {
        isHealthy: boolean;
        lastError?: string;
    };
    destroy(): void;
}
