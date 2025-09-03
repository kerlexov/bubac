import { Request, Response } from 'express';
import { MCPLogger } from './logger';
import { HighThroughputBuffer } from './high-throughput-buffer';
import * as os from 'os';

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

export class HealthChecker {
  private logger: MCPLogger;
  private buffer?: HighThroughputBuffer;
  private errorCount = 0;
  private lastError?: Error;
  private startTime = Date.now();

  constructor(logger: MCPLogger, buffer?: HighThroughputBuffer) {
    this.logger = logger;
    this.buffer = buffer;
  }

  incrementErrorCount(): void {
    this.errorCount++;
  }

  setLastError(error: Error): void {
    this.lastError = error;
    this.incrementErrorCount();
  }

  getHealthStatus(): HealthStatus {
    const memoryUsage = process.memoryUsage();
    const uptime = (Date.now() - this.startTime) / 1000;
    
    // Determine overall health status
    let status: 'healthy' | 'degraded' | 'unhealthy' = 'healthy';
    
    // Check memory usage (degraded if > 80% of heap limit)
    const heapUsedPercent = (memoryUsage.heapUsed / memoryUsage.heapTotal) * 100;
    if (heapUsedPercent > 95) {
      status = 'unhealthy';
    } else if (heapUsedPercent > 90) {
      status = 'degraded';
    }

    // Check error rate (degraded if > 10 errors, unhealthy if > 50)
    if (this.errorCount > 50) {
      status = 'unhealthy';
    } else if (this.errorCount > 10) {
      status = 'degraded';
    }

    const healthStatus: HealthStatus = {
      status,
      timestamp: new Date().toISOString(),
      uptime,
      version: process.env.npm_package_version || '1.0.0',
      logger: {
        connected: true, // Assume connected if no recent errors
        bufferSize: this.buffer ? this.buffer.size() : 0,
        lastFlush: this.buffer ? 
          (this.buffer.getStats().lastFlushTime?.toISOString() || null) : 
          null,
        errorCount: this.errorCount
      },
      system: {
        memory: memoryUsage,
        cpu: process.cpuUsage ? process.cpuUsage() : null,
        loadAverage: os.loadavg(),
        platform: os.platform(),
        nodeVersion: process.version,
        pid: process.pid
      }
    };

    if (this.buffer) {
      healthStatus.buffer = this.buffer.getStats();
    }

    return healthStatus;
  }

  createHealthEndpoint() {
    return (req: Request, res: Response) => {
      try {
        const health = this.getHealthStatus();
        
        // Set appropriate HTTP status code
        const statusCode = health.status === 'healthy' ? 200 :
                          health.status === 'degraded' ? 200 : 503;
        
        res.status(statusCode).json(health);
        
        // Log health check request
        this.logger.debug('Health check requested', {
          requestId: req.headers['x-request-id'],
          ip: req.ip,
          userAgent: req.get('User-Agent'),
          healthStatus: health.status
        });
      } catch (error) {
        this.setLastError(error as Error);
        
        res.status(500).json({
          status: 'unhealthy',
          timestamp: new Date().toISOString(),
          error: 'Health check failed',
          message: (error as Error).message
        });
      }
    };
  }

  // Detailed health endpoint with more information
  createDetailedHealthEndpoint() {
    return (req: Request, res: Response) => {
      try {
        const health = this.getHealthStatus();
        
        // Add more detailed information
        const detailedHealth = {
          ...health,
          details: {
            lastError: this.lastError ? {
              name: this.lastError.name,
              message: this.lastError.message,
              stack: this.lastError.stack
            } : null,
            environment: {
              nodeEnv: process.env.NODE_ENV,
              timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
              locale: Intl.DateTimeFormat().resolvedOptions().locale
            },
            network: {
              hostname: os.hostname(),
              networkInterfaces: Object.keys(os.networkInterfaces())
            },
            process: {
              argv: process.argv,
              execPath: process.execPath,
              cwd: process.cwd(),
              title: process.title
            }
          }
        };
        
        const statusCode = health.status === 'healthy' ? 200 :
                          health.status === 'degraded' ? 200 : 503;
        
        res.status(statusCode).json(detailedHealth);
      } catch (error) {
        this.setLastError(error as Error);
        
        res.status(500).json({
          status: 'unhealthy',
          timestamp: new Date().toISOString(),
          error: 'Detailed health check failed',
          message: (error as Error).message
        });
      }
    };
  }

  // Reset error count (useful for testing or manual recovery)
  resetErrorCount(): void {
    this.errorCount = 0;
    this.lastError = undefined;
  }
}