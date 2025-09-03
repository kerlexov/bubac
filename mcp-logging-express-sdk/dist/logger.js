"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.MCPLogger = void 0;
const axios = require('axios');
const { v4: uuidv4 } = require('uuid');
const types_1 = require("./types");
const buffer_1 = require("./buffer");
const high_throughput_buffer_1 = require("./high-throughput-buffer");
const health_check_1 = require("./health-check");
class MCPLogger {
    constructor(config) {
        this.isShuttingDown = false;
        this.config = {
            bufferSize: 1000,
            flushInterval: 5000,
            retryConfig: {
                maxRetries: 3,
                initialDelay: 1000,
                maxDelay: 30000,
                backoffMultiplier: 2
            },
            enableHealthCheck: false,
            healthCheckPort: 3001,
            ...config
        };
        this.httpClient = axios.create({
            baseURL: this.config.serverUrl,
            timeout: 10000,
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': `mcp-logging-express-sdk/1.0.0`
            }
        });
        this.buffer = new buffer_1.Buffer(this.config.bufferSize);
        // Use high-throughput buffer for better performance if buffer size is large
        if (this.config.bufferSize > 500) {
            this.highThroughputBuffer = new high_throughput_buffer_1.HighThroughputBuffer(Math.min(this.config.bufferSize, 1000), Math.max(Math.floor(this.config.bufferSize / 1000), 5));
        }
        this.healthChecker = new health_check_1.HealthChecker(this, this.highThroughputBuffer);
        this.startFlushTimer();
        this.setupGracefulShutdown();
    }
    debug(message, metadata) {
        this.log(types_1.LogLevel.DEBUG, message, metadata);
    }
    info(message, metadata) {
        this.log(types_1.LogLevel.INFO, message, metadata);
    }
    warn(message, metadata) {
        this.log(types_1.LogLevel.WARN, message, metadata);
    }
    error(message, metadata) {
        this.log(types_1.LogLevel.ERROR, message, metadata);
    }
    fatal(message, metadata) {
        this.log(types_1.LogLevel.FATAL, message, metadata);
    }
    log(level, message, metadata) {
        if (this.isShuttingDown) {
            return;
        }
        const logEntry = {
            id: uuidv4(),
            timestamp: new Date(),
            level,
            message,
            serviceName: this.config.serviceName,
            agentId: this.config.agentId,
            platform: 'express',
            metadata: {
                ...metadata,
                nodeVersion: process.version,
                pid: process.pid
            },
            deviceInfo: {
                platform: 'Server',
                version: process.version,
                model: 'Node.js',
                appVersion: process.env.npm_package_version || '1.0.0'
            },
            sourceLocation: this.getSourceLocation()
        };
        // Add to buffer (non-blocking)
        if (this.highThroughputBuffer) {
            this.highThroughputBuffer.add(logEntry);
        }
        else {
            this.buffer.add(logEntry);
        }
    }
    getSourceLocation() {
        const stack = new Error().stack;
        if (!stack)
            return undefined;
        const lines = stack.split('\n');
        // Skip the first few lines (Error, getSourceLocation, log method)
        const callerLine = lines[4];
        if (!callerLine)
            return undefined;
        const match = callerLine.match(/at\s+(.+?)\s+\((.+):(\d+):(\d+)\)/);
        if (match) {
            return {
                function: match[1],
                file: match[2],
                line: parseInt(match[3])
            };
        }
        return undefined;
    }
    startFlushTimer() {
        this.flushTimer = setInterval(() => {
            this.flush().catch(err => {
                console.error('Failed to flush logs:', err.message);
            });
        }, this.config.flushInterval);
    }
    async flush() {
        const logs = this.highThroughputBuffer ?
            this.highThroughputBuffer.flush() :
            this.buffer.flush();
        if (logs.length === 0) {
            return;
        }
        try {
            await this.sendLogs(logs);
        }
        catch (error) {
            // Re-add logs to buffer for retry
            if (this.highThroughputBuffer) {
                logs.forEach(log => this.highThroughputBuffer.add(log));
            }
            else {
                logs.forEach(log => this.buffer.add(log));
            }
            // Track error in health checker
            if (this.healthChecker) {
                this.healthChecker.setLastError(error);
            }
            throw error;
        }
    }
    async sendLogs(logs) {
        const { maxRetries, initialDelay, maxDelay, backoffMultiplier } = this.config.retryConfig;
        let lastError;
        for (let attempt = 0; attempt <= maxRetries; attempt++) {
            try {
                await this.httpClient.post('/api/logs', { logs });
                return;
            }
            catch (error) {
                lastError = error;
                if (attempt === maxRetries) {
                    break;
                }
                // Calculate delay with exponential backoff
                const delay = Math.min(initialDelay * Math.pow(backoffMultiplier, attempt), maxDelay);
                await new Promise(resolve => setTimeout(resolve, delay));
            }
        }
        throw lastError;
    }
    setupGracefulShutdown() {
        this.shutdownHandler = async () => {
            this.isShuttingDown = true;
            if (this.flushTimer) {
                clearInterval(this.flushTimer);
            }
            try {
                await this.flush();
            }
            catch (error) {
                console.error('Failed to flush logs during shutdown:', error);
            }
        };
        process.on('SIGINT', this.shutdownHandler);
        process.on('SIGTERM', this.shutdownHandler);
        process.on('beforeExit', this.shutdownHandler);
    }
    getHealthChecker() {
        return this.healthChecker;
    }
    getBufferStats() {
        if (this.highThroughputBuffer) {
            return this.highThroughputBuffer.getStats();
        }
        return {
            size: this.buffer.size(),
            isFull: this.buffer.isFull()
        };
    }
    async close() {
        this.isShuttingDown = true;
        if (this.flushTimer) {
            clearInterval(this.flushTimer);
        }
        // Remove event listeners to prevent memory leaks
        if (this.shutdownHandler) {
            process.removeListener('SIGINT', this.shutdownHandler);
            process.removeListener('SIGTERM', this.shutdownHandler);
            process.removeListener('beforeExit', this.shutdownHandler);
        }
        await this.flush();
    }
}
exports.MCPLogger = MCPLogger;
//# sourceMappingURL=logger.js.map