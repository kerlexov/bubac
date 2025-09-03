import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import React from 'react';
import { MCPErrorBoundary, withMCPErrorBoundary } from '../error-boundary';
import { MCPLogger } from '../types';

// Component that throws an error
const ThrowingComponent: React.FC<{ shouldThrow?: boolean }> = ({ shouldThrow = true }) => {
  if (shouldThrow) {
    throw new Error('Test error');
  }
  return <div>No error</div>;
};

describe('MCPErrorBoundary', () => {
  let mockLogger: MCPLogger;

  beforeEach(() => {
    mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
      fatal: vi.fn(),
      logPerformance: vi.fn(),
      logUserInteraction: vi.fn(),
      flush: vi.fn(),
      getHealthStatus: vi.fn(() => ({ isHealthy: true })),
    };

    // Suppress console.error for error boundary tests
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  it('should catch and log errors', () => {
    render(
      <MCPErrorBoundary logger={mockLogger}>
        <ThrowingComponent />
      </MCPErrorBoundary>
    );

    expect(mockLogger.error).toHaveBeenCalledWith(
      'React Error Boundary caught error',
      expect.objectContaining({
        error: 'Test error',
        stack: expect.any(String),
        componentStack: expect.any(String),
        errorBoundary: true,
      })
    );
  });

  it('should render default error UI when error occurs', () => {
    render(
      <MCPErrorBoundary logger={mockLogger}>
        <ThrowingComponent />
      </MCPErrorBoundary>
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('An error occurred in this component. The error has been logged.')).toBeInTheDocument();
  });

  it('should render custom fallback UI when provided', () => {
    const customFallback = <div>Custom error message</div>;

    render(
      <MCPErrorBoundary logger={mockLogger} fallback={customFallback}>
        <ThrowingComponent />
      </MCPErrorBoundary>
    );

    expect(screen.getByText('Custom error message')).toBeInTheDocument();
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();
  });

  it('should call custom error handler when provided', () => {
    const onError = vi.fn();

    render(
      <MCPErrorBoundary logger={mockLogger} onError={onError}>
        <ThrowingComponent />
      </MCPErrorBoundary>
    );

    expect(onError).toHaveBeenCalledWith(
      expect.any(Error),
      expect.objectContaining({
        componentStack: expect.any(String),
      })
    );
  });

  it('should render children normally when no error occurs', () => {
    render(
      <MCPErrorBoundary logger={mockLogger}>
        <ThrowingComponent shouldThrow={false} />
      </MCPErrorBoundary>
    );

    expect(screen.getByText('No error')).toBeInTheDocument();
    expect(mockLogger.error).not.toHaveBeenCalled();
  });

  it('should show error details in expandable section', () => {
    render(
      <MCPErrorBoundary logger={mockLogger}>
        <ThrowingComponent />
      </MCPErrorBoundary>
    );

    expect(screen.getByText('Error details')).toBeInTheDocument();
  });
});

describe('withMCPErrorBoundary HOC', () => {
  let mockLogger: MCPLogger;

  beforeEach(() => {
    mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
      fatal: vi.fn(),
      logPerformance: vi.fn(),
      logUserInteraction: vi.fn(),
      flush: vi.fn(),
      getHealthStatus: vi.fn(() => ({ isHealthy: true })),
    };

    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  it('should wrap component with error boundary', () => {
    const WrappedComponent = withMCPErrorBoundary(ThrowingComponent, mockLogger);

    render(<WrappedComponent />);

    expect(mockLogger.error).toHaveBeenCalled();
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('should use custom fallback when provided', () => {
    const customFallback = <div>HOC custom error</div>;
    const WrappedComponent = withMCPErrorBoundary(ThrowingComponent, mockLogger, customFallback);

    render(<WrappedComponent />);

    expect(screen.getByText('HOC custom error')).toBeInTheDocument();
  });

  it('should pass props to wrapped component', () => {
    const TestComponent: React.FC<{ message: string }> = ({ message }) => (
      <div>{message}</div>
    );

    const WrappedComponent = withMCPErrorBoundary(TestComponent, mockLogger);

    render(<WrappedComponent message="Hello World" />);

    expect(screen.getByText('Hello World')).toBeInTheDocument();
  });

  it('should set correct display name', () => {
    const TestComponent: React.FC = () => <div>Test</div>;
    TestComponent.displayName = 'TestComponent';

    const WrappedComponent = withMCPErrorBoundary(TestComponent, mockLogger);

    expect(WrappedComponent.displayName).toBe('withMCPErrorBoundary(TestComponent)');
  });
});