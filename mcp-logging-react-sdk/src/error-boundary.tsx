import React, { Component, ErrorInfo, ReactNode } from 'react';
import { MCPLogger } from './types';

interface Props {
  logger: MCPLogger;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

export class MCPErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    const { logger, onError } = this.props;

    // Log the error through MCP Logger
    logger.error('React Error Boundary caught error', {
      error: error.message,
      stack: error.stack,
      componentStack: errorInfo.componentStack,
      errorBoundary: true,
    });

    // Call custom error handler if provided
    if (onError) {
      onError(error, errorInfo);
    }
  }

  render() {
    if (this.state.hasError) {
      // Render fallback UI
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div style={{ 
          padding: '20px', 
          border: '1px solid #ff6b6b', 
          borderRadius: '4px',
          backgroundColor: '#ffe0e0',
          color: '#d63031'
        }}>
          <h2>Something went wrong</h2>
          <p>An error occurred in this component. The error has been logged.</p>
          <details style={{ marginTop: '10px' }}>
            <summary>Error details</summary>
            <pre style={{ 
              marginTop: '10px', 
              padding: '10px', 
              backgroundColor: '#f8f8f8',
              overflow: 'auto',
              fontSize: '12px'
            }}>
              {this.state.error?.stack}
            </pre>
          </details>
        </div>
      );
    }

    return this.props.children;
  }
}

// HOC for wrapping components with error boundary
export function withMCPErrorBoundary<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  logger: MCPLogger,
  fallback?: ReactNode
) {
  const WithErrorBoundary = (props: P) => (
    <MCPErrorBoundary logger={logger} fallback={fallback}>
      <WrappedComponent {...props} />
    </MCPErrorBoundary>
  );

  WithErrorBoundary.displayName = `withMCPErrorBoundary(${WrappedComponent.displayName || WrappedComponent.name})`;
  
  return WithErrorBoundary;
}