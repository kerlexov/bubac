import React, { useState, useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { 
  MCPLoggerProvider, 
  useMCPLogger, 
  MCPErrorBoundary,
  BrowserFeatures,
  LocalStorageManager 
} from '../src';

// Component to demonstrate browser features
const BrowserFeaturesDemo: React.FC = () => {
  const logger = useMCPLogger();
  const [storageStats, setStorageStats] = useState<any>(null);
  const [browserEnv, setBrowserEnv] = useState<any>(null);

  useEffect(() => {
    // Get browser environment info
    const browserFeatures = new BrowserFeatures(logger);
    const env = browserFeatures.getBrowserEnvironment();
    setBrowserEnv(env);

    // Get storage stats
    const stats = logger.getStorageStats();
    setStorageStats(stats);

    // Log component mount with browser info
    logger.info('Browser features demo mounted', {
      userAgent: env.userAgent,
      platform: env.platform,
      screenResolution: env.screenResolution,
      language: env.language,
      onLine: env.onLine,
    });

    // Cleanup
    return () => {
      browserFeatures.destroy();
    };
  }, [logger]);

  const triggerPerformanceTest = () => {
    const startTime = performance.now();
    
    // Simulate heavy computation
    let result = 0;
    for (let i = 0; i < 100000; i++) {
      result += Math.sqrt(i);
    }
    
    const endTime = performance.now();
    const duration = endTime - startTime;
    
    logger.logPerformance({
      customMetric: duration,
    });
    
    logger.info('Performance test completed', {
      duration: Math.round(duration),
      result: Math.round(result),
    });
  };

  const triggerNetworkTest = async () => {
    logger.info('Starting network test');
    
    try {
      // Test successful request
      const response = await fetch('https://httpbin.org/json');
      const data = await response.json();
      
      logger.info('Network test successful', {
        status: response.status,
        dataSize: JSON.stringify(data).length,
      });
    } catch (error) {
      logger.error('Network test failed', {
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  };

  const triggerStorageTest = () => {
    const storageManager = new LocalStorageManager({
      keyPrefix: 'demo-storage',
      maxEntries: 10,
      maxSizeBytes: 1024,
    });

    // Create test logs
    const testLogs = Array.from({ length: 5 }, (_, i) => ({
      id: `test-${i}`,
      timestamp: new Date(),
      level: 'INFO' as const,
      message: `Test storage message ${i}`,
      serviceName: 'demo-service',
      agentId: 'demo-agent',
      platform: 'react',
    }));

    // Save and load logs
    const saved = storageManager.saveLogs(testLogs);
    const loaded = storageManager.loadLogs();
    const stats = storageManager.getStorageStats();

    logger.info('Storage test completed', {
      saved,
      loadedCount: loaded.length,
      storageStats: stats,
    });

    // Update display
    setStorageStats(logger.getStorageStats());
  };

  const triggerVisibilityTest = () => {
    logger.info('Visibility test - try switching tabs or minimizing window');
    
    // Log current visibility state
    logger.logUserInteraction({
      type: 'custom',
      timestamp: new Date(),
      metadata: {
        action: 'visibility_test_triggered',
        currentlyVisible: !document.hidden,
        visibilityState: document.visibilityState,
      },
    });
  };

  const triggerOfflineTest = () => {
    logger.info('Offline test - try disconnecting your internet');
    
    // Force some logs that might fail to send
    for (let i = 0; i < 3; i++) {
      logger.info(`Offline test message ${i + 1}`, {
        timestamp: new Date().toISOString(),
        testId: `offline-${i}`,
      });
    }
    
    // Force flush to trigger potential network errors
    logger.flush().then(() => {
      logger.info('Flush completed successfully');
    }).catch((error) => {
      logger.error('Flush failed', { error: error.message });
    });
  };

  return (
    <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
      <h1>Browser Features Demo</h1>
      
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '20px' }}>
        <div style={{ padding: '15px', border: '1px solid #ccc', borderRadius: '5px' }}>
          <h3>Browser Environment</h3>
          {browserEnv && (
            <div style={{ fontSize: '12px', fontFamily: 'monospace' }}>
              <div><strong>Platform:</strong> {browserEnv.platform}</div>
              <div><strong>Language:</strong> {browserEnv.language}</div>
              <div><strong>Screen:</strong> {browserEnv.screenResolution}</div>
              <div><strong>Online:</strong> {browserEnv.onLine ? 'Yes' : 'No'}</div>
              <div><strong>Cookies:</strong> {browserEnv.cookieEnabled ? 'Enabled' : 'Disabled'}</div>
              <div><strong>CPU Cores:</strong> {browserEnv.hardwareConcurrency}</div>
              {browserEnv.deviceMemory && (
                <div><strong>Memory:</strong> {browserEnv.deviceMemory}GB</div>
              )}
              {browserEnv.connection && (
                <div><strong>Connection:</strong> {browserEnv.connection.effectiveType}</div>
              )}
            </div>
          )}
        </div>

        <div style={{ padding: '15px', border: '1px solid #ccc', borderRadius: '5px' }}>
          <h3>Storage Statistics</h3>
          {storageStats && (
            <div style={{ fontSize: '12px', fontFamily: 'monospace' }}>
              <div><strong>Entries:</strong> {storageStats.entryCount}/{storageStats.maxEntries}</div>
              <div><strong>Size:</strong> {storageStats.sizeBytes} bytes</div>
              <div><strong>Usage:</strong> {storageStats.usagePercent}%</div>
              <div style={{ 
                width: '100%', 
                height: '10px', 
                backgroundColor: '#f0f0f0', 
                marginTop: '5px',
                borderRadius: '5px'
              }}>
                <div style={{
                  width: `${storageStats.usagePercent}%`,
                  height: '100%',
                  backgroundColor: storageStats.usagePercent > 80 ? '#ff6b6b' : '#51cf66',
                  borderRadius: '5px'
                }} />
              </div>
            </div>
          )}
        </div>
      </div>

      <div style={{ marginBottom: '20px' }}>
        <h3>Feature Tests</h3>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px' }}>
          <button 
            onClick={triggerPerformanceTest}
            style={{ padding: '10px 15px', backgroundColor: '#007bff', color: 'white', border: 'none', borderRadius: '5px' }}
          >
            Performance Test
          </button>
          
          <button 
            onClick={triggerNetworkTest}
            style={{ padding: '10px 15px', backgroundColor: '#28a745', color: 'white', border: 'none', borderRadius: '5px' }}
          >
            Network Test
          </button>
          
          <button 
            onClick={triggerStorageTest}
            style={{ padding: '10px 15px', backgroundColor: '#ffc107', color: 'black', border: 'none', borderRadius: '5px' }}
          >
            Storage Test
          </button>
          
          <button 
            onClick={triggerVisibilityTest}
            style={{ padding: '10px 15px', backgroundColor: '#6f42c1', color: 'white', border: 'none', borderRadius: '5px' }}
          >
            Visibility Test
          </button>
          
          <button 
            onClick={triggerOfflineTest}
            style={{ padding: '10px 15px', backgroundColor: '#fd7e14', color: 'white', border: 'none', borderRadius: '5px' }}
          >
            Offline Test
          </button>
        </div>
      </div>

      <div style={{ padding: '15px', backgroundColor: '#e9ecef', borderRadius: '5px' }}>
        <h3>🔍 What's Being Monitored</h3>
        <ul style={{ margin: 0, paddingLeft: '20px' }}>
          <li><strong>Performance Metrics:</strong> Page load time, Core Web Vitals (LCP, CLS, FID), long tasks</li>
          <li><strong>User Interactions:</strong> Clicks, form submissions, scrolling, navigation</li>
          <li><strong>Browser Events:</strong> Visibility changes, focus/blur, resize, online/offline</li>
          <li><strong>Error Capture:</strong> Unhandled errors, promise rejections, React errors</li>
          <li><strong>Console Capture:</strong> All console.log, console.warn, console.error calls</li>
          <li><strong>Local Storage:</strong> Offline buffering with automatic retry and cleanup</li>
          <li><strong>Network Monitoring:</strong> Connection status changes, request failures</li>
        </ul>
      </div>

      <div style={{ marginTop: '20px', padding: '15px', backgroundColor: '#d1ecf1', borderRadius: '5px' }}>
        <h3>📊 Check Your Developer Tools</h3>
        <p>Open your browser's developer tools to see:</p>
        <ul style={{ margin: 0, paddingLeft: '20px' }}>
          <li><strong>Console:</strong> All logs are being captured and sent to the server</li>
          <li><strong>Network:</strong> POST requests to the logging server with batched logs</li>
          <li><strong>Application → Local Storage:</strong> Buffered logs during offline scenarios</li>
          <li><strong>Performance:</strong> Core Web Vitals and custom performance metrics</li>
        </ul>
      </div>
    </div>
  );
};

// Root component
const BrowserFeaturesRoot: React.FC = () => {
  const config = {
    serverUrl: 'http://localhost:9080',
    serviceName: 'browser-features-demo',
    agentId: 'browser-demo-001',
    bufferSize: 3, // Small buffer for frequent flushes
    flushInterval: 2000, // Flush every 2 seconds
    enableConsoleCapture: true,
    enableErrorBoundary: true,
    enablePerformanceMetrics: true,
    enableUserInteractions: true,
    enableLocalStorage: true,
    logLevel: 'DEBUG' as const,
  };

  return (
    <MCPLoggerProvider config={config}>
      <MCPErrorBoundary 
        logger={{} as any} // Will be provided by context
        fallback={
          <div style={{ padding: '20px', backgroundColor: '#f8d7da', color: '#721c24' }}>
            <h2>🚨 Error Boundary Activated</h2>
            <p>An error occurred and was logged to the MCP server.</p>
          </div>
        }
      >
        <BrowserFeaturesDemo />
      </MCPErrorBoundary>
    </MCPLoggerProvider>
  );
};

// Mount the app
const container = document.getElementById('root');
if (container) {
  const root = createRoot(container);
  root.render(<BrowserFeaturesRoot />);
}