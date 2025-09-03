"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ErrorHandler = void 0;
class ErrorHandler {
    constructor(logger) {
        this.logger = logger;
    }
    enableGlobalErrorCapture() {
        // Capture uncaught exceptions
        this.originalUncaughtException = process.listeners('uncaughtException')[0];
        process.on('uncaughtException', (error) => {
            this.logger.fatal('Uncaught Exception', {
                error: {
                    name: error.name,
                    message: error.message,
                    stack: error.stack
                },
                process: {
                    pid: process.pid,
                    uptime: process.uptime(),
                    memoryUsage: process.memoryUsage()
                }
            });
            // Call original handler if it exists
            if (this.originalUncaughtException) {
                this.originalUncaughtException(error);
            }
        });
        // Capture unhandled promise rejections
        this.originalUnhandledRejection = process.listeners('unhandledRejection')[0];
        process.on('unhandledRejection', (reason, promise) => {
            this.logger.error('Unhandled Promise Rejection', {
                reason: reason instanceof Error ? {
                    name: reason.name,
                    message: reason.message,
                    stack: reason.stack
                } : reason,
                promise: promise.toString(),
                process: {
                    pid: process.pid,
                    uptime: process.uptime(),
                    memoryUsage: process.memoryUsage()
                }
            });
            // Call original handler if it exists
            if (this.originalUnhandledRejection) {
                this.originalUnhandledRejection(reason, promise);
            }
        });
        // Capture process warnings
        this.originalWarning = process.listeners('warning')[0];
        process.on('warning', (warning) => {
            this.logger.warn('Process Warning', {
                warning: {
                    name: warning.name,
                    message: warning.message,
                    stack: warning.stack
                },
                process: {
                    pid: process.pid,
                    uptime: process.uptime()
                }
            });
            // Call original handler if it exists
            if (this.originalWarning) {
                this.originalWarning(warning);
            }
        });
    }
    disableGlobalErrorCapture() {
        // Remove our listeners and restore originals
        process.removeAllListeners('uncaughtException');
        process.removeAllListeners('unhandledRejection');
        process.removeAllListeners('warning');
        if (this.originalUncaughtException) {
            process.on('uncaughtException', this.originalUncaughtException);
        }
        if (this.originalUnhandledRejection) {
            process.on('unhandledRejection', this.originalUnhandledRejection);
        }
        if (this.originalWarning) {
            process.on('warning', this.originalWarning);
        }
    }
}
exports.ErrorHandler = ErrorHandler;
//# sourceMappingURL=error-handler.js.map