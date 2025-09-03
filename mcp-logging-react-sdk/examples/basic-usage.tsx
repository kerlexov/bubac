import React, { useState, useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { 
  MCPLoggerProvider, 
  useMCPLogger, 
  MCPErrorBoundary,
  withMCPErrorBoundary 
} from '../src';

// Basic component using the logger
const UserProfile: React.FC<{ userId: string }> = ({ userId }) => {
  const logger = useMCPLogger();
  const [user, setUser] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const fetchUser = async () => {
    setLoading(true);
    logger.info('Fetching user profile', { userId });

    try {
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      const userData = { id: userId, name: 'John Doe', email: 'john@example.com' };
      setUser(userData);
      
      logger.info('User profile loaded successfully', { userId, user: userData });
    } catch (error) {
      logger.error('Failed to fetch user profile', { userId, error: error.message });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUser();
  }, [userId]);

  const handleClick = () => {
    logger.info('User profile clicked', { userId });
    // The click will also be automatically tracked by user interaction monitoring
  };

  if (loading) {
    return <div>Loading user profile...</div>;
  }

  return (
    <div onClick={handleClick} style={{ padding: '20px', border: '1px solid #ccc', margin: '10px' }}>
      <h3>User Profile</h3>
      {user ? (
        <div>
          <p>Name: {user.name}</p>
          <p>Email: {user.email}</p>
        </div>
      ) : (
        <p>No user data</p>
      )}
    </div>
  );
};

// Component that demonstrates error boundary
const ErrorProneComponent: React.FC<{ shouldError: boolean }> = ({ shouldError }) => {
  const logger = useMCPLogger();

  useEffect(() => {
    logger.info('ErrorProneComponent mounted', { shouldError });
  }, [logger, shouldError]);

  if (shouldError) {
    throw new Error('Intentional error for demonstration');
  }

  return (
    <div style={{ padding: '20px', backgroundColor: '#e8f5e8', margin: '10px' }}>
      <h3>Error Prone Component</h3>
      <p>This component is working fine!</p>
    </div>
  );
};

// Component wrapped with HOC error boundary
const WrappedErrorComponent = withMCPErrorBoundary(
  ErrorProneComponent,
  // Note: In real usage, you'd get the logger from context
  {} as any, // Placeholder for demo
  <div style={{ padding: '20px', backgroundColor: '#ffe8e8', color: '#d63031' }}>
    HOC Error Boundary: Something went wrong!
  </div>
);

// Main app component
const App: React.FC = () => {
  const logger = useMCPLogger();
  const [shouldError, setShouldError] = useState(false);
  const [userId, setUserId] = useState('user-123');

  useEffect(() => {
    logger.info('App component mounted');
    
    // Log some performance metrics
    logger.logPerformance({
      pageLoadTime: 1500,
      firstContentfulPaint: 800,
      largestContentfulPaint: 1200,
    });

    // Log a custom user interaction
    logger.logUserInteraction({
      type: 'custom',
      timestamp: new Date(),
      metadata: { action: 'app_initialized' },
    });
  }, [logger]);

  const handleConsoleTest = () => {
    console.log('This console.log will be captured automatically');
    console.warn('This console.warn will be captured automatically');
    console.error('This console.error will be captured automatically');
  };

  const handleManualLog = () => {
    logger.debug('Debug message with metadata', { feature: 'manual_logging' });
    logger.info('Info message', { timestamp: new Date().toISOString() });
    logger.warn('Warning message', { level: 'warning' });
    logger.error('Error message', { errorCode: 'DEMO_ERROR' });
  };

  const handleFlush = async () => {
    logger.info('Manual flush requested');
    await logger.flush();
    alert('Logs flushed to server!');
  };

  const handleHealthCheck = () => {
    const health = logger.getHealthStatus();
    alert(`Logger Health: ${health.isHealthy ? 'Healthy' : 'Unhealthy'}\nLast Error: ${health.lastError || 'None'}`);
  };

  return (
    <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
      <h1>MCP Logging React SDK Demo</h1>
      
      <div style={{ marginBottom: '20px' }}>
        <h2>Controls</h2>
        <button onClick={handleConsoleTest} style={{ margin: '5px' }}>
          Test Console Capture
        </button>
        <button onClick={handleManualLog} style={{ margin: '5px' }}>
          Manual Logging
        </button>
        <button onClick={handleFlush} style={{ margin: '5px' }}>
          Flush Logs
        </button>
        <button onClick={handleHealthCheck} style={{ margin: '5px' }}>
          Health Check
        </button>
        <button 
          onClick={() => setShouldError(!shouldError)} 
          style={{ margin: '5px', backgroundColor: shouldError ? '#ff6b6b' : '#51cf66' }}
        >
          {shouldError ? 'Fix Error' : 'Trigger Error'}
        </button>
      </div>

      <div>
        <h2>User ID</h2>
        <input 
          value={userId} 
          onChange={(e) => setUserId(e.target.value)}
          style={{ padding: '5px', margin: '5px' }}
        />
      </div>

      <div>
        <h2>Components</h2>
        
        <UserProfile userId={userId} />
        
        <MCPErrorBoundary 
          logger={logger}
          fallback={
            <div style={{ padding: '20px', backgroundColor: '#ffe8e8', color: '#d63031', margin: '10px' }}>
              <h3>Error Boundary Caught Error</h3>
              <p>The component below threw an error and was caught by the error boundary.</p>
            </div>
          }
        >
          <ErrorProneComponent shouldError={shouldError} />
        </MCPErrorBoundary>
      </div>

      <div style={{ marginTop: '20px', padding: '20px', backgroundColor: '#f8f9fa' }}>
        <h3>Automatic Features Active</h3>
        <ul>
          <li>✅ Console log capture (try the console test button)</li>
          <li>✅ Error boundary integration (try the error trigger button)</li>
          <li>✅ Performance metrics (logged on app start)</li>
          <li>✅ User interaction tracking (clicks are automatically logged)</li>
          <li>✅ Local storage buffering (for offline scenarios)</li>
        </ul>
      </div>
    </div>
  );
};

// Root component with provider
const Root: React.FC = () => {
  const config = {
    serverUrl: 'http://localhost:8080',
    serviceName: 'react-demo-app',
    agentId: 'web-demo-001',
    bufferSize: 10,
    flushInterval: 3000,
    enableConsoleCapture: true,
    enableErrorBoundary: true,
    enablePerformanceMetrics: true,
    enableUserInteractions: true,
    enableLocalStorage: true,
    logLevel: 'DEBUG' as const,
  };

  return (
    <MCPLoggerProvider config={config}>
      <App />
    </MCPLoggerProvider>
  );
};

// Mount the app
const container = document.getElementById('root');
if (container) {
  const root = createRoot(container);
  root.render(<Root />);
}