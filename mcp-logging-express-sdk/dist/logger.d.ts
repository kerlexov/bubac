import { MCPLoggerConfig, LoggerAdapter } from './types';
import { HealthChecker } from './health-check';
export declare class MCPLogger implements LoggerAdapter {
    private httpClient;
    private buffer;
    private highThroughputBuffer?;
    private config;
    private flushTimer?;
    private isShuttingDown;
    private healthChecker?;
    constructor(config: MCPLoggerConfig);
    debug(message: string, metadata?: Record<string, any>): void;
    info(message: string, metadata?: Record<string, any>): void;
    warn(message: string, metadata?: Record<string, any>): void;
    error(message: string, metadata?: Record<string, any>): void;
    fatal(message: string, metadata?: Record<string, any>): void;
    private log;
    private getSourceLocation;
    private startFlushTimer;
    private flush;
    private sendLogs;
    private shutdownHandler?;
    private setupGracefulShutdown;
    getHealthChecker(): HealthChecker | undefined;
    getBufferStats(): any;
    close(): Promise<void>;
}
//# sourceMappingURL=logger.d.ts.map