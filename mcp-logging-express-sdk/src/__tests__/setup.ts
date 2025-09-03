// Jest setup file
import 'jest';

// Increase timeout for async operations
jest.setTimeout(10000);

// Mock console methods to avoid noise in tests
global.console = {
  ...console,
  error: jest.fn(),
  warn: jest.fn(),
  log: jest.fn(),
  info: jest.fn(),
  debug: jest.fn(),
};