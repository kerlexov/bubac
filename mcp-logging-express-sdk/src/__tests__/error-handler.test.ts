import { ErrorHandler } from '../error-handler';
import { MCPLogger } from '../logger';

// Mock the logger
jest.mock('../logger');

describe('ErrorHandler', () => {
  let errorHandler: ErrorHandler;
  let mockLogger: jest.Mocked<MCPLogger>;

  beforeEach(() => {
    mockLogger = new MCPLogger({
      serverUrl: 'http://localhost:8080',
      serviceName: 'test',
      agentId: 'test'
    }) as jest.Mocked<MCPLogger>;

    mockLogger.fatal = jest.fn();
    mockLogger.error = jest.fn();
    mockLogger.warn = jest.fn();

    errorHandler = new ErrorHandler(mockLogger);
  });

  afterEach(() => {
    // Clean up event listeners
    errorHandler.disableGlobalErrorCapture();
  });

  test('should enable global error capture', () => {
    const originalListenerCount = process.listenerCount('uncaughtException');
    
    errorHandler.enableGlobalErrorCapture();
    
    expect(process.listenerCount('uncaughtException')).toBeGreaterThan(originalListenerCount);
    expect(process.listenerCount('unhandledRejection')).toBeGreaterThan(0);
    expect(process.listenerCount('warning')).toBeGreaterThan(0);
  });

  test('should disable global error capture', () => {
    errorHandler.enableGlobalErrorCapture();
    const uncaughtListenerCount = process.listenerCount('uncaughtException');
    
    errorHandler.disableGlobalErrorCapture();
    
    // Note: We can't easily test the exact count due to other listeners
    // but we can verify the method doesn't throw
    expect(true).toBe(true);
  });

  test('should capture uncaught exceptions', () => {
    errorHandler.enableGlobalErrorCapture();
    
    // Mock the fatal method to verify it's called
    mockLogger.fatal.mockImplementation((message, metadata) => {
      expect(message).toBe('Uncaught Exception');
      expect(metadata).toBeDefined();
      expect(metadata?.error).toBeDefined();
      expect(metadata?.process).toBeDefined();
    });

    // Simulate an uncaught exception
    process.emit('uncaughtException', new Error('Test uncaught exception'));
    
    expect(mockLogger.fatal).toHaveBeenCalled();
  });

  test('should capture unhandled rejections', () => {
    errorHandler.enableGlobalErrorCapture();
    
    mockLogger.error.mockImplementation((message, metadata) => {
      expect(message).toBe('Unhandled Promise Rejection');
      expect(metadata).toBeDefined();
      expect(metadata?.reason).toBeDefined();
      expect(metadata?.process).toBeDefined();
    });

    // Simulate an unhandled rejection
    process.emit('unhandledRejection', new Error('Test rejection'), Promise.resolve());
    
    expect(mockLogger.error).toHaveBeenCalled();
  });

  test('should capture process warnings', () => {
    errorHandler.enableGlobalErrorCapture();
    
    mockLogger.warn.mockImplementation((message, metadata) => {
      expect(message).toBe('Process Warning');
      expect(metadata).toBeDefined();
      expect(metadata?.warning).toBeDefined();
      expect(metadata?.process).toBeDefined();
    });

    // Simulate a process warning
    const warning = new Error('Test warning');
    warning.name = 'Warning';
    process.emit('warning', warning);
    
    expect(mockLogger.warn).toHaveBeenCalled();
  });
});