import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { createRoot } from 'react-dom/client';
import { 
  MCPLoggerProvider, 
  useMCPLogger, 
  useMCPLoggerConfig,
  MCPErrorBoundary,
  withMCPErrorBoundary 
} from '../src';

// Custom hook for API calls with logging
const useApiCall = <T,>(url: string, dependencies: any[] = []) => {
  const logger = useMCPLogger();
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    const startTime = performance.now();
    setLoading(true);
    setError(null);
    
    logger.info('API call started', { url, method: 'GET' });

    try {
      // Simulate API call with random delay and occasional failures
      await new Promise(resolve => setTimeout(resolve, Math.random() * 2000 + 500));
      
      if (Math.random() < 0.2) { // 20% chance of failure
        throw new Error('Simulated API failure');
      }

      const mockData = { 
        id: Math.random().toString(36).substr(2, 9),
        timestamp: new Date().toISOString(),
        url 
      } as T;
      
      setData(mockData);
      
      const endTime = performance.now();
      const duration = endTime - startTime;
      
      logger.info('API call successful', { 
        url, 
        duration: Math.round(duration),
        dataSize: JSON.stringify(mockData).length 
      });
      
      logger.logPerformance({
        customMetric: duration,
      });
      
    } catch (err) {
      const endTime = performance.now();
      const duration = endTime - startTime;
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';
      
      setError(errorMessage);
      logger.error('API call failed', { 
        url, 
        error: errorMessage,
        duration: Math.round(duration)
      });
    } finally {
      setLoading(false);
    }
  }, [url, logger, ...dependencies]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch: fetchData };
};

// Component with performance monitoring
const PerformanceMonitoredComponent: React.FC = () => {
  const logger = useMCPLogger();
  const [renderCount, setRenderCount] = useState(0);
  const [heavyComputation, setHeavyComputation] = useState<number | null>(null);

  // Track renders
  useEffect(() => {
    setRenderCount(prev => prev + 1);
    logger.debug('Component rendered', { renderCount: renderCount + 1 });
  });

  // Simulate heavy computation with performance tracking
  const performHeavyComputation = useCallback(() => {
    const startTime = performance.now();
    logger.info('Starting heavy computation');

    // Simulate CPU-intensive task
    let result = 0;
    for (let i = 0; i < 1000000; i++) {
      result += Math.sqrt(i);
    }

    const endTime = performance.now();
    const duration = endTime - startTime;
    
    setHeavyComputation(result);
    
    logger.info('Heavy computation completed', { 
      duration: Math.round(duration),
      result: Math.round(result)
    });
    
    logger.logPerformance({
      customMetric: duration,
    });
  }, [logger]);

  return (
    <div style={{ padding: '20px', border: '2px solid #007bff', margin: '10px' }}>
      <h3>Performance Monitored Component</h3>
      <p>Render count: {renderCount}</p>
      <button onClick={performHeavyComputation}>
        Run Heavy Computation
      </button>
      {heavyComputation && (
        <p>Computation result: {Math.round(heavyComputation)}</p>
      )}
    </div>
  );
};

// Component with user interaction tracking
const InteractionTracker: React.FC = () => {
  const logger = useMCPLogger();
  const [interactions, setInteractions] = useState<string[]>([]);

  const logCustomInteraction = (type: string, details: any) => {
    const interaction = `${type}: ${JSON.stringify(details)}`;
    setInteractions(prev => [...prev.slice(-4), interaction]); // Keep last 5
    
    logger.logUserInteraction({
      type: 'custom',
      timestamp: new Date(),
      metadata: { customType: type, ...details },
    });
  };

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const formData = new FormData(e.target as HTMLFormElement);
    const data = Object.fromEntries(formData.entries());
    
    logCustomInteraction('form_submit', { 
      formId: 'demo-form',
      fields: Object.keys(data),
      hasEmail: !!data.email 
    });
  };

  const handleHover = () => {
    logCustomInteraction('hover', { element: 'hover-button' });
  };

  const handleDoubleClick = () => {
    logCustomInteraction('double_click', { element: 'double-click-button' });
  };

  return (
    <div style={{ padding: '20px', border: '2px solid #28a745', margin: '10px' }}>
      <h3>Interaction Tracker</h3>
      
      <form onSubmit={handleFormSubmit} style={{ marginBottom: '15px' }}>
        <input 
          name="email" 
          type="email" 
          placeholder="Email" 
          style={{ margin: '5px', padding: '5px' }}
        />
        <input 
          name="name" 
          type="text" 
          placeholder="Name" 
          style={{ margin: '5px', padding: '5px' }}
        />
        <button type="submit" style={{ margin: '5px', padding: '5px' }}>
          Submit Form
        </button>
      </form>

      <div>
        <button 
          onMouseEnter={handleHover}
          style={{ margin: '5px', padding: '5px' }}
        >
          Hover Me
        </button>
        <button 
          onDoubleClick={handleDoubleClick}
          style={{ margin: '5px', padding: '5px' }}
        >
          Double Click Me
        </button>
      </div>

      <div style={{ marginTop: '15px' }}>
        <h4>Recent Custom Interactions:</h4>
        <ul style={{ fontSize: '12px', maxHeight: '100px', overflow: 'auto' }}>
          {interactions.map((interaction, index) => (
            <li key={index}>{interaction}</li>
          ))}
        </ul>
      </div>
    </div>
  );
};

// Component demonstrating error scenarios
const ErrorScenarios: React.FC = () => {
  const logger = useMCPLogger();
  const [errorType, setErrorType] = useState<string>('none');

  const triggerError = (type: string) => {
    setErrorType(type);
    
    switch (type) {
      case 'sync':
        logger.warn('About to throw synchronous error');
        throw new Error('Synchronous error for testing');
      
      case 'async':
        logger.warn('About to trigger async error');
        setTimeout(() => {
          throw new Error('Asynchronous error for testing');
        }, 100);
        break;
      
      case 'promise':
        logger.warn('About to trigger promise rejection');
        Promise.reject(new Error('Promise rejection for testing'));
        break;
      
      case 'network':
        logger.warn('Simulating network error');
        fetch('/non-existent-endpoint')
          .catch(err => logger.error('Network error caught', { error: err.message }));
        break;
      
      default:
        setErrorType('none');
    }
  };

  if (errorType === 'sync') {
    throw new Error('Synchronous error for testing');
  }

  return (
    <div style={{ padding: '20px', border: '2px solid #dc3545', margin: '10px' }}>
      <h3>Error Scenarios</h3>
      <p>Test different types of errors and their logging:</p>
      
      <div>
        <button onClick={() => triggerError('sync')} style={{ margin: '5px', backgroundColor: '#dc3545', color: 'white' }}>
          Sync Error (Caught by Boundary)
        </button>
        <button onClick={() => triggerError('async')} style={{ margin: '5px', backgroundColor: '#fd7e14', color: 'white' }}>
          Async Error (Global Handler)
        </button>
        <button onClick={() => triggerError('promise')} style={{ margin: '5px', backgroundColor: '#6f42c1', color: 'white' }}>
          Promise Rejection
        </button>
        <button onClick={() => triggerError('network')} style={{ margin: '5px', backgroundColor: '#20c997', color: 'white' }}>
          Network Error
        </button>
      </div>
    </div>
  );
};

// Component showing configuration usage
const ConfigDisplay: React.FC = () => {
  const config = useMCPLoggerConfig();
  const logger = useMCPLogger();

  const configEntries = useMemo(() => 
    Object.entries(config).map(([key, value]) => ({
      key,
      value: typeof value === 'object' ? JSON.stringify(value) : String(value)
    })), [config]
  );

  useEffect(() => {
    logger.info('Config display mounted', { configKeys: Object.keys(config) });
  }, [logger, config]);

  return (
    <div style={{ padding: '20px', border: '2px solid #6c757d', margin: '10px' }}>
      <h3>Current Configuration</h3>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            <th style={{ border: '1px solid #ccc', padding: '8px', textAlign: 'left' }}>Key</th>
            <th style={{ border: '1px solid #ccc', padding: '8px', textAlign: 'left' }}>Value</th>
          </tr>
        </thead>
        <tbody>
          {configEntries.map(({ key, value }) => (
            <tr key={key}>
              <td style={{ border: '1px solid #ccc', padding: '8px', fontFamily: 'monospace' }}>{key}</td>
              <td style={{ border: '1px solid #ccc', padding: '8px', fontFamily: 'monospace', fontSize: '12px' }}>
                {value.length > 50 ? `${value.substring(0, 50)}...` : value}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

// Main app with all advanced features
const AdvancedApp: React.FC = () => {
  const logger = useMCPLogger();
  const { data, loading, error, refetch } = useApiCall<any>('/api/demo-data');
  const [showErrors, setShowErrors] = useState(false);

  useEffect(() => {
    logger.info('Advanced app initialized', {
      features: [
        'custom_hooks',
        'performance_monitoring', 
        'interaction_tracking',
        'error_scenarios',
        'config_display'
      ]
    });
  }, [logger]);

  return (
    <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
      <h1>MCP Logging React SDK - Advanced Features</h1>
      
      <div style={{ marginBottom: '20px', padding: '15px', backgroundColor: '#e9ecef' }}>
        <h2>API Call with Custom Hook</h2>
        {loading && <p>Loading...</p>}
        {error && <p style={{ color: 'red' }}>Error: {error}</p>}
        {data && (
          <div>
            <p>Data loaded: {JSON.stringify(data)}</p>
            <button onClick={refetch}>Refetch Data</button>
          </div>
        )}
      </div>

      <PerformanceMonitoredComponent />
      
      <InteractionTracker />
      
      <div>
        <button 
          onClick={() => setShowErrors(!showErrors)}
          style={{ margin: '10px', padding: '10px' }}
        >
          {showErrors ? 'Hide' : 'Show'} Error Scenarios
        </button>
        
        {showErrors && (
          <MCPErrorBoundary 
            logger={logger}
            fallback={
              <div style={{ padding: '20px', backgroundColor: '#f8d7da', color: '#721c24', margin: '10px' }}>
                <h3>🚨 Error Boundary Active</h3>
                <p>An error was caught and logged. The component has been replaced with this fallback UI.</p>
                <button onClick={() => setShowErrors(false)}>Reset</button>
              </div>
            }
            onError={(error, errorInfo) => {
              console.log('Custom error handler called:', error.message);
            }}
          >
            <ErrorScenarios />
          </MCPErrorBoundary>
        )}
      </div>

      <ConfigDisplay />

      <div style={{ marginTop: '30px', padding: '20px', backgroundColor: '#d1ecf1', color: '#0c5460' }}>
        <h3>🔍 Check Your Browser Console & Network Tab</h3>
        <ul>
          <li>Console logs are being captured automatically</li>
          <li>Network requests to the logging server are visible in Network tab</li>
          <li>All user interactions (clicks, hovers, etc.) are being logged</li>
          <li>Performance metrics are being collected and sent</li>
          <li>Errors are being caught and logged with full context</li>
        </ul>
      </div>
    </div>
  );
};

// Root with advanced configuration
const AdvancedRoot: React.FC = () => {
  const config = {
    serverUrl: 'http://localhost:9080',
    serviceName: 'react-advanced-demo',
    agentId: 'web-advanced-001',
    bufferSize: 5, // Smaller buffer for more frequent flushes
    flushInterval: 2000, // Flush every 2 seconds
    retryAttempts: 5,
    retryDelay: 500,
    enableConsoleCapture: true,
    enableErrorBoundary: true,
    enablePerformanceMetrics: true,
    enableUserInteractions: true,
    enableLocalStorage: true,
    logLevel: 'DEBUG' as const,
  };

  return (
    <MCPLoggerProvider config={config}>
      <AdvancedApp />
    </MCPLoggerProvider>
  );
};

// Mount the advanced app
const container = document.getElementById('root');
if (container) {
  const root = createRoot(container);
  root.render(<AdvancedRoot />);
}