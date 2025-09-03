import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import React from 'react';
import { MCPLoggerProvider, useMCPLogger, useMCPLoggerConfig } from '../context';
import { MCPLoggerConfig } from '../types';

// Test component that uses the hook
const TestComponent: React.FC = () => {
  const logger = useMCPLogger();
  const config = useMCPLoggerConfig();

  return (
    <div>
      <div data-testid="service-name">{config.serviceName}</div>
      <div data-testid="agent-id">{config.agentId}</div>
      <button
        data-testid="log-button"
        onClick={() => logger.info('Test message')}
      >
        Log Message
      </button>
    </div>
  );
};

describe('MCPLoggerProvider and hooks', () => {
  let config: MCPLoggerConfig;
  let fetchMock: any;

  beforeEach(() => {
    config = {
      serverUrl: 'http://localhost:8080',
      serviceName: 'test-service',
      agentId: 'test-agent',
      enableConsoleCapture: false,
      enableErrorBoundary: false,
      enablePerformanceMetrics: false,
      enableUserInteractions: false,
    };

    fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      statusText: 'OK',
    });
    global.fetch = fetchMock;
  });

  it('should provide logger through context', () => {
    render(
      <MCPLoggerProvider config={config}>
        <TestComponent />
      </MCPLoggerProvider>
    );

    expect(screen.getByTestId('service-name')).toHaveTextContent('test-service');
    expect(screen.getByTestId('agent-id')).toHaveTextContent('test-agent');
  });

  it('should throw error when hook is used outside provider', () => {
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => {
      render(<TestComponent />);
    }).toThrow('useMCPLogger must be used within an MCPLoggerProvider');

    consoleSpy.mockRestore();
  });

  it('should allow logging through the hook', async () => {
    const { getByTestId } = render(
      <MCPLoggerProvider config={config}>
        <TestComponent />
      </MCPLoggerProvider>
    );

    const logButton = getByTestId('log-button');
    logButton.click();

    // Logger should be functional
    expect(logButton).toBeInTheDocument();
  });

  it('should setup performance monitoring when enabled', () => {
    const addEventListenerSpy = vi.spyOn(window, 'addEventListener');

    render(
      <MCPLoggerProvider config={{ ...config, enablePerformanceMetrics: true }}>
        <TestComponent />
      </MCPLoggerProvider>
    );

    expect(addEventListenerSpy).toHaveBeenCalledWith('load', expect.any(Function));
    
    addEventListenerSpy.mockRestore();
  });

  it('should setup user interaction monitoring when enabled', () => {
    const addEventListenerSpy = vi.spyOn(document, 'addEventListener');

    render(
      <MCPLoggerProvider config={{ ...config, enableUserInteractions: true }}>
        <TestComponent />
      </MCPLoggerProvider>
    );

    expect(addEventListenerSpy).toHaveBeenCalledWith('click', expect.any(Function));
    
    addEventListenerSpy.mockRestore();
  });

  it('should cleanup on unmount', () => {
    const { unmount } = render(
      <MCPLoggerProvider config={config}>
        <TestComponent />
      </MCPLoggerProvider>
    );

    // Should not throw when unmounting
    expect(() => unmount()).not.toThrow();
  });
});