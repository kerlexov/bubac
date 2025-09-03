"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.HealthChecker = void 0;
const os = __importStar(require("os"));
class HealthChecker {
    constructor(logger, buffer) {
        this.errorCount = 0;
        this.startTime = Date.now();
        this.logger = logger;
        this.buffer = buffer;
    }
    incrementErrorCount() {
        this.errorCount++;
    }
    setLastError(error) {
        this.lastError = error;
        this.incrementErrorCount();
    }
    getHealthStatus() {
        var _a;
        const memoryUsage = process.memoryUsage();
        const uptime = (Date.now() - this.startTime) / 1000;
        // Determine overall health status
        let status = 'healthy';
        // Check memory usage (degraded if > 80% of heap limit)
        const heapUsedPercent = (memoryUsage.heapUsed / memoryUsage.heapTotal) * 100;
        if (heapUsedPercent > 90) {
            status = 'unhealthy';
        }
        else if (heapUsedPercent > 80) {
            status = 'degraded';
        }
        // Check error rate (degraded if > 10 errors, unhealthy if > 50)
        if (this.errorCount > 50) {
            status = 'unhealthy';
        }
        else if (this.errorCount > 10) {
            status = 'degraded';
        }
        const healthStatus = {
            status,
            timestamp: new Date().toISOString(),
            uptime,
            version: process.env.npm_package_version || '1.0.0',
            logger: {
                connected: true,
                bufferSize: this.buffer ? this.buffer.size() : 0,
                lastFlush: this.buffer ?
                    (((_a = this.buffer.getStats().lastFlushTime) === null || _a === void 0 ? void 0 : _a.toISOString()) || null) :
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
        return (req, res) => {
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
            }
            catch (error) {
                this.setLastError(error);
                res.status(500).json({
                    status: 'unhealthy',
                    timestamp: new Date().toISOString(),
                    error: 'Health check failed',
                    message: error.message
                });
            }
        };
    }
    // Detailed health endpoint with more information
    createDetailedHealthEndpoint() {
        return (req, res) => {
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
            }
            catch (error) {
                this.setLastError(error);
                res.status(500).json({
                    status: 'unhealthy',
                    timestamp: new Date().toISOString(),
                    error: 'Detailed health check failed',
                    message: error.message
                });
            }
        };
    }
    // Reset error count (useful for testing or manual recovery)
    resetErrorCount() {
        this.errorCount = 0;
        this.lastError = undefined;
    }
}
exports.HealthChecker = HealthChecker;
//# sourceMappingURL=health-check.js.map