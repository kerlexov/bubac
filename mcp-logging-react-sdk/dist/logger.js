export class MCPLoggerImpl {
    constructor(config) {
        this.buffer = [];
        this.isHealthy = true;
        this.retryCount = 0;
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
        this.startFlushTimer();
        this.setupConsoleCapture();
        this.setupErrorCapture();
        this.loadBufferedLogs();
    }
    debug(message, metadata) {
        this.log('DEBUG', message, metadata);
    }
    info(message, metadata) {
        this.log('INFO', message, metadata);
    }
    warn(message, metadata) {
        this.log('WARN', message, metadata);
    }
    error(message, metadata) {
        this.log('ERROR', message, metadata);
    }
    fatal(message, metadata) {
        this.log('FATAL', message, metadata);
    }
    logPerformance(metrics) {
        this.log('INFO', 'Performance metrics captured', {
            type: 'performance',
            metrics
        });
    }
    logUserInteraction(interaction) {
        this.log('INFO', `User interaction: ${interaction.type}`, {
            type: 'user_interaction',
            interaction
        });
    }
    log(level, message, metadata) {
        if (!this.shouldLog(level)) {
            return;
        }
        const entry = {
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
    shouldLog(level) {
        const levels = ['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'];
        const currentLevelIndex = levels.indexOf(this.config.logLevel);
        const logLevelIndex = levels.indexOf(level);
        return logLevelIndex >= currentLevelIndex;
    }
    async flush() {
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
        }
        catch (error) {
            this.handleSendError(error, logsToSend);
        }
    }
    async sendLogs(logs) {
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
    handleSendError(error, logs) {
        this.isHealthy = false;
        this.lastError = error.message || 'Unknown error';
        if (this.config.enableLocalStorage) {
            this.saveToLocalStorage(logs);
        }
        if (this.retryCount < this.config.retryAttempts) {
            this.retryCount++;
            const delay = this.config.retryDelay * Math.pow(2, this.retryCount - 1);
            setTimeout(() => {
                this.sendLogs(logs).catch(() => {
                    // Final retry failed, logs are lost or saved to localStorage
                });
            }, delay);
        }
    }
    saveToLocalStorage(logs) {
        try {
            const existingLogs = localStorage.getItem('mcp-logger-buffer');
            const bufferedLogs = existingLogs ? JSON.parse(existingLogs) : [];
            const updatedLogs = [...bufferedLogs, ...logs];
            // Keep only the most recent 1000 logs to prevent localStorage overflow
            const trimmedLogs = updatedLogs.slice(-1000);
            localStorage.setItem('mcp-logger-buffer', JSON.stringify(trimmedLogs));
        }
        catch (error) {
            console.warn('Failed to save logs to localStorage:', error);
        }
    }
    loadBufferedLogs() {
        if (!this.config.enableLocalStorage) {
            return;
        }
        try {
            const bufferedLogs = localStorage.getItem('mcp-logger-buffer');
            if (bufferedLogs) {
                const logs = JSON.parse(bufferedLogs);
                this.buffer.push(...logs);
                localStorage.removeItem('mcp-logger-buffer');
                // Attempt to send buffered logs
                if (logs.length > 0) {
                    this.flush();
                }
            }
        }
        catch (error) {
            console.warn('Failed to load buffered logs from localStorage:', error);
        }
    }
    startFlushTimer() {
        this.flushTimer = setInterval(() => {
            this.flush();
        }, this.config.flushInterval);
    }
    setupConsoleCapture() {
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
    setupErrorCapture() {
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
    getDeviceInfo() {
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
    getSourceLocation() {
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
        }
        catch (error) {
            // Ignore errors in source location detection
        }
        return undefined;
    }
    generateId() {
        return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    }
    getHealthStatus() {
        return {
            isHealthy: this.isHealthy,
            lastError: this.lastError,
        };
    }
    destroy() {
        if (this.flushTimer) {
            clearInterval(this.flushTimer);
        }
        this.flush();
    }
}
