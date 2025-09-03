import { MCPLogger } from './logger';

export class ErrorHandler {
  private logger: MCPLogger;
  private originalUncaughtException?: (error: Error) => void;
  private originalUnhandledRejection?: (reason: any, promise: Promise<any>) => void;
  private originalWarning?: (warning: Error) => void;

  constructor(logger: MCPLogger) {
    this.logger = logger;
  }

  enableGlobalErrorCapture(): void {
    // Capture uncaught exceptions
    this.originalUncaughtException = process.listeners('uncaughtException')[0] as any;
    process.on('uncaughtException', (error: Error) => {
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
    this.originalUnhandledRejection = process.listeners('unhandledRejection')[0] as any;
    process.on('unhandledRejection', (reason: any, promise: Promise<any>) => {
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
    this.originalWarning = process.listeners('warning')[0] as any;
    process.on('warning', (warning: Error) => {
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

  disableGlobalErrorCapture(): void {
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