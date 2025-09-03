/// <reference types="node" />
import { Request, Response } from 'express';
import { MCPLogger } from './logger';
import { HighThroughputBuffer } from './high-throughput-buffer';
export interface HealthStatus {
    status: 'healthy' | 'degraded' | 'unhealthy';
    timestamp: string;
    uptime: number;
    version: string;
    logger: {
        connected: boolean;
        bufferSize: number;
        lastFlush: string | null;
        errorCount: number;
    };
    system: {
        memory: NodeJS.MemoryUsage;
        cpu: NodeJS.CpuUsage | null;
        loadAverage: number[];
        platform: string;
        nodeVersion: string;
        pid: number;
    };
    buffer?: any;
}
export declare class HealthChecker {
    private logger;
    private buffer?;
    private errorCount;
    private lastError?;
    private startTime;
    constructor(logger: MCPLogger, buffer?: HighThroughputBuffer);
    incrementErrorCount(): void;
    setLastError(error: Error): void;
    getHealthStatus(): HealthStatus;
    createHealthEndpoint(): (req: Request, res: Response) => void;
    createDetailedHealthEndpoint(): (req: Request, res: Response) => void;
    resetErrorCount(): void;
}
//# sourceMappingURL=health-check.d.ts.map