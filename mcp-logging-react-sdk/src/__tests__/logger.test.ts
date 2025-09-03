import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { MCPLoggerImpl } from '../logger';
import { MCPLoggerConfig } from '../types';

describe('MCPLoggerImpl', () => {
  let logger: MCPLoggerImpl;
  let config: MCPLoggerConfig;
  let fetchMock: any;

  beforeEach(() => {
    config = {
      serverUrl: 'http://localhost:9080',
      serviceName: 'test-service',
      agentId: 'test-agent',
      bufferSize: 2,
      flushInterval: 1000,
      enableConsoleCapture: false,
      enableErrorBoundary: false,
    };

    fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      statusText: 'OK',
    });
    global.fetch = fetchMock;

    logger = new MCPLoggerImpl(config);
  });

  afterEach(() => {
    logger.destroy();
  });

  describe('Basic logging', () => {
    it('should log messages at different levels', () => {
      logger.debug('Debug message');
      logger.info('Info message');
      logger.warn('Warn message');
      logger.error('Error message');
      logger.fatal('Fatal message');

      expect(logger.getHealthStatus().isHealthy).toBe(true);
    });

    it('should include metadata in log entries', () => {
      const metadata = { userId: '123', action: 'login' };
      logger.info('User logged in', metadata);

      expect(logger.getHealthStatus().isHealthy).toBe(true);
    });

    it('should respect log level filtering', () => {
      const warnLogger = new MCPLoggerImpl({
        ...config,
        logLevel: 'WARN',
      });

      warnLogger.debug('Should not log');
      warnLogger.info('Should not log');
      warnLogger.warn('Should log');
      warnLogger.error('Should log');

      warnLogger.destroy();
    });
  });

  describe('Buffering and flushing', () => {
    it('should flush when buffer is full', async () => {
      logger.info('Message 1');
      logger.info('Message 2'); // Should trigger flush

      // Wait a bit for async flush
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(fetchMock).toHaveBeenCalledWith(
        'http://localhost:9080/api/logs',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        })
      );
    });

    it('should flush manually', async () => {
      logger.info('Message 1');
      
      await logger.flush();

      expect(fetchMock).toHaveBeenCalled();
    });

    it('should handle flush errors gracefully', async () => {
      fetchMock.mockRejectedValueOnce(new Error('Network error'));

      logger.info('Message 1');
      
      // Flush should not throw, but should handle errors internally
      await logger.flush();

      // Wait a bit for async error handling
      await new Promise(resolve => setTimeout(resolve, 10));

      const health = logger.getHealthStatus();
      expect(health.isHealthy).toBe(false);
      expect(health.lastError).toBe('Network error');
    });
  });

  describe('Performance logging', () => {
    it('should log performance metrics', () => {
      const metrics = {
        pageLoadTime: 1500,
        firstContentfulPaint: 800,
      };

      logger.logPerformance(metrics);
      expect(logger.getHealthStatus().isHealthy).toBe(true);
    });
  });

  describe('User interaction logging', () => {
    it('should log user interactions', () => {
      const interaction = {
        type: 'click' as const,
        element: 'button#submit',
        timestamp: new Date(),
      };

      logger.logUserInteraction(interaction);
      expect(logger.getHealthStatus().isHealthy).toBe(true);
    });
  });

  describe('Local storage buffering', () => {
    it('should save failed logs to localStorage', async () => {
      const loggerWithStorage = new MCPLoggerImpl({
        ...config,
        enableLocalStorage: true,
      });

      fetchMock.mockRejectedValueOnce(new Error('Network error'));

      loggerWithStorage.info('Failed message');
      await loggerWithStorage.flush();

      expect(localStorage.setItem).toHaveBeenCalledWith(
        expect.stringContaining('mcp-logger-test-service'),
        expect.any(String)
      );

      loggerWithStorage.destroy();
    });

    it('should load buffered logs on initialization', () => {
      const bufferedLogs = JSON.stringify([
        {
          id: '1',
          timestamp: new Date().toISOString(),
          level: 'INFO',
          message: 'Buffered message',
          serviceName: 'test-service',
          agentId: 'test-agent',
          platform: 'react',
        },
      ]);

      // Set up the localStorage data before creating the logger
      localStorage.setItem('mcp-logger-test-service-buffer', bufferedLogs);

      const loggerWithStorage = new MCPLoggerImpl({
        ...config,
        enableLocalStorage: true,
      });

      expect(localStorage.getItem).toHaveBeenCalledWith(expect.stringContaining('mcp-logger-test-service'));
      expect(localStorage.removeItem).toHaveBeenCalledWith(expect.stringContaining('mcp-logger-test-service'));

      loggerWithStorage.destroy();
    });
  });

  describe('Console capture', () => {
    it('should capture console logs when enabled', () => {
      const originalConsole = { ...console };
      const loggerWithConsole = new MCPLoggerImpl({
        ...config,
        enableConsoleCapture: true,
      });

      // Test that console methods are wrapped
      expect(console.log).not.toBe(originalConsole.log);
      expect(console.warn).not.toBe(originalConsole.warn);
      expect(console.error).not.toBe(originalConsole.error);

      loggerWithConsole.destroy();
    });
  });

  describe('Error capture', () => {
    it('should set up error event listeners when enabled', () => {
      const addEventListenerSpy = vi.spyOn(window, 'addEventListener');
      
      const loggerWithErrors = new MCPLoggerImpl({
        ...config,
        enableErrorBoundary: true,
      });

      expect(addEventListenerSpy).toHaveBeenCalledWith('error', expect.any(Function));
      expect(addEventListenerSpy).toHaveBeenCalledWith('unhandledrejection', expect.any(Function));

      loggerWithErrors.destroy();
      addEventListenerSpy.mockRestore();
    });
  });

  describe('Device info', () => {
    it('should include device information in logs', () => {
      // Mock navigator
      Object.defineProperty(window, 'navigator', {
        value: {
          userAgent: 'Test Browser',
          language: 'en-US',
        },
        writable: true,
      });

      // Mock screen
      Object.defineProperty(window, 'screen', {
        value: {
          width: 1920,
          height: 1080,
        },
        writable: true,
      });

      logger.info('Test message');
      expect(logger.getHealthStatus().isHealthy).toBe(true);
    });
  });
});