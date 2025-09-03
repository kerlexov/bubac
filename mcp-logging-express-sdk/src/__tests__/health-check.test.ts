import { HealthChecker } from '../health-check';
import { MCPLogger } from '../logger';
import { HighThroughputBuffer } from '../high-throughput-buffer';

// Mock the logger
jest.mock('../logger');

describe('HealthChecker', () => {
  let healthChecker: HealthChecker;
  let mockLogger: jest.Mocked<MCPLogger>;
  let mockBuffer: HighThroughputBuffer;

  beforeEach(() => {
    mockLogger = new MCPLogger({
      serverUrl: 'http://localhost:8080',
      serviceName: 'test',
      agentId: 'test'
    }) as jest.Mocked<MCPLogger>;

    mockBuffer = new HighThroughputBuffer(100, 5);
    healthChecker = new HealthChecker(mockLogger, mockBuffer);
  });

  test('should return healthy status initially', () => {
    const status = healthChecker.getHealthStatus();
    
    expect(status.status).toBe('healthy');
    expect(status.logger.errorCount).toBe(0);
    expect(status.uptime).toBeGreaterThanOrEqual(0);
  });

  test('should track errors', () => {
    const error = new Error('Test error');
    healthChecker.setLastError(error);

    const status = healthChecker.getHealthStatus();
    expect(status.logger.errorCount).toBe(1);
  });

  test('should become degraded with many errors', () => {
    // Add many errors
    for (let i = 0; i < 15; i++) {
      healthChecker.incrementErrorCount();
    }

    const status = healthChecker.getHealthStatus();
    expect(status.status).toBe('degraded');
  });

  test('should become unhealthy with too many errors', () => {
    // Add too many errors
    for (let i = 0; i < 55; i++) {
      healthChecker.incrementErrorCount();
    }

    const status = healthChecker.getHealthStatus();
    expect(status.status).toBe('unhealthy');
  });

  test('should reset error count', () => {
    healthChecker.incrementErrorCount();
    healthChecker.incrementErrorCount();
    
    expect(healthChecker.getHealthStatus().logger.errorCount).toBe(2);
    
    healthChecker.resetErrorCount();
    
    expect(healthChecker.getHealthStatus().logger.errorCount).toBe(0);
  });

  test('should create health endpoint', () => {
    const endpoint = healthChecker.createHealthEndpoint();
    expect(typeof endpoint).toBe('function');
  });

  test('should create detailed health endpoint', () => {
    const endpoint = healthChecker.createDetailedHealthEndpoint();
    expect(typeof endpoint).toBe('function');
  });

  test('should include buffer stats when available', () => {
    const status = healthChecker.getHealthStatus();
    
    expect(status.buffer).toBeDefined();
    expect(status.buffer.totalItems).toBeDefined();
    expect(status.buffer.bufferSize).toBeDefined();
  });

  test('should include system information', () => {
    const status = healthChecker.getHealthStatus();
    
    expect(status.system.memory).toBeDefined();
    expect(status.system.platform).toBeDefined();
    expect(status.system.nodeVersion).toBeDefined();
    expect(status.system.pid).toBeDefined();
  });
});