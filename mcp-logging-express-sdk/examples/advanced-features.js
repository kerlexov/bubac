const express = require('express');
const { 
  MCPLogger, 
  createMiddleware, 
  ErrorHandler, 
  HighThroughputBuffer 
} = require('../dist/index');

const app = express();
app.use(express.json());

// Create logger with high-throughput buffer
const logger = new MCPLogger({
  serverUrl: 'http://localhost:9080',
  serviceName: 'advanced-example',
  agentId: 'advanced-001',
  bufferSize: 2000, // Large buffer triggers high-throughput mode
  flushInterval: 1000,
  enableHealthCheck: true,
  healthCheckPort: 3002
});

// Set up global error capture
const errorHandler = new ErrorHandler(logger);
errorHandler.enableGlobalErrorCapture();

// Add comprehensive middleware
app.use(createMiddleware({
  serverUrl: 'http://localhost:9080',
  serviceName: 'advanced-example',
  agentId: 'advanced-001',
  logRequests: true,
  logResponses: true,
  includeHeaders: true,
  includeBody: true,
  excludePaths: ['/health', '/health/detailed'],
  sensitiveHeaders: ['authorization', 'x-api-key', 'cookie']
}));

// Health check endpoints
const healthChecker = logger.getHealthChecker();
if (healthChecker) {
  app.get('/health', healthChecker.createHealthEndpoint());
  app.get('/health/detailed', healthChecker.createDetailedHealthEndpoint());
}

// Buffer stats endpoint
app.get('/stats/buffer', (req, res) => {
  const stats = logger.getBufferStats();
  res.json({
    buffer: stats,
    timestamp: new Date().toISOString()
  });
});

// High-load endpoint for testing throughput
app.post('/api/high-load', (req, res) => {
  const { count = 100, delay = 0 } = req.body;
  
  logger.info('High-load test started', { count, delay });
  
  // Generate many log entries quickly
  for (let i = 0; i < count; i++) {
    setTimeout(() => {
      logger.info(`High-load message ${i}`, {
        iteration: i,
        timestamp: new Date().toISOString(),
        randomData: Math.random().toString(36).substring(7),
        metadata: {
          batch: Math.floor(i / 10),
          isEven: i % 2 === 0,
          progress: (i / count * 100).toFixed(2) + '%'
        }
      });
    }, delay * i);
  }
  
  res.json({
    message: `Generating ${count} log entries`,
    estimatedDuration: count * delay,
    startTime: new Date().toISOString()
  });
});

// Error simulation endpoints
app.get('/api/simulate/uncaught-exception', (req, res) => {
  logger.warn('Simulating uncaught exception');
  res.json({ message: 'Uncaught exception will be thrown in 1 second' });
  
  setTimeout(() => {
    throw new Error('Simulated uncaught exception');
  }, 1000);
});

app.get('/api/simulate/unhandled-rejection', (req, res) => {
  logger.warn('Simulating unhandled promise rejection');
  res.json({ message: 'Unhandled rejection will occur in 1 second' });
  
  setTimeout(() => {
    Promise.reject(new Error('Simulated unhandled rejection'));
  }, 1000);
});

app.get('/api/simulate/memory-pressure', (req, res) => {
  logger.warn('Simulating memory pressure');
  
  // Create memory pressure by generating large objects
  const largeObjects = [];
  for (let i = 0; i < 1000; i++) {
    largeObjects.push({
      id: i,
      data: new Array(10000).fill(`data-${i}`),
      timestamp: new Date().toISOString()
    });
    
    if (i % 100 === 0) {
      logger.info(`Created ${i} large objects`, {
        memoryUsage: process.memoryUsage(),
        objectCount: largeObjects.length
      });
    }
  }
  
  res.json({
    message: 'Memory pressure created',
    objectsCreated: largeObjects.length,
    memoryUsage: process.memoryUsage()
  });
  
  // Clean up after a delay
  setTimeout(() => {
    largeObjects.length = 0;
    if (global.gc) {
      global.gc();
    }
    logger.info('Memory pressure cleaned up');
  }, 5000);
});

// Structured logging examples
app.post('/api/user/:userId/action', (req, res) => {
  const { userId } = req.params;
  const { action, data } = req.body;
  
  const requestId = req.headers['x-request-id'] || `req-${Date.now()}`;
  
  logger.info('User action started', {
    requestId,
    userId,
    action,
    userAgent: req.get('User-Agent'),
    ip: req.ip,
    timestamp: new Date().toISOString()
  });
  
  try {
    // Simulate processing
    const processingTime = Math.random() * 1000;
    
    setTimeout(() => {
      const success = Math.random() > 0.1; // 90% success rate
      
      if (success) {
        logger.info('User action completed successfully', {
          requestId,
          userId,
          action,
          processingTime,
          result: 'success'
        });
        
        res.json({
          success: true,
          action,
          processingTime,
          timestamp: new Date().toISOString()
        });
      } else {
        const error = new Error('Simulated processing error');
        
        logger.error('User action failed', {
          requestId,
          userId,
          action,
          processingTime,
          error: {
            name: error.name,
            message: error.message,
            stack: error.stack
          }
        });
        
        res.status(500).json({
          success: false,
          error: error.message,
          action,
          processingTime
        });
      }
    }, processingTime);
    
  } catch (error) {
    logger.error('User action exception', {
      requestId,
      userId,
      action,
      error: {
        name: error.name,
        message: error.message,
        stack: error.stack
      }
    });
    
    res.status(500).json({
      success: false,
      error: 'Internal server error'
    });
  }
});

// Performance monitoring endpoint
app.get('/api/performance', (req, res) => {
  const startTime = process.hrtime.bigint();
  
  // Simulate some work
  let sum = 0;
  for (let i = 0; i < 1000000; i++) {
    sum += Math.sqrt(i);
  }
  
  const endTime = process.hrtime.bigint();
  const duration = Number(endTime - startTime) / 1000000; // Convert to milliseconds
  
  const performanceData = {
    duration,
    sum,
    memoryUsage: process.memoryUsage(),
    cpuUsage: process.cpuUsage(),
    uptime: process.uptime(),
    loadAverage: require('os').loadavg()
  };
  
  logger.info('Performance test completed', performanceData);
  
  res.json({
    message: 'Performance test completed',
    ...performanceData,
    timestamp: new Date().toISOString()
  });
});

// Error handling middleware
app.use((err, req, res, next) => {
  const requestId = req.headers['x-request-id'] || 'unknown';
  
  logger.error('Express error handler', {
    requestId,
    error: {
      name: err.name,
      message: err.message,
      stack: err.stack
    },
    request: {
      method: req.method,
      url: req.url,
      headers: req.headers,
      body: req.body
    }
  });
  
  res.status(500).json({
    error: 'Internal server error',
    requestId,
    timestamp: new Date().toISOString()
  });
});

const PORT = process.env.PORT || 3002;

app.listen(PORT, () => {
  logger.info('Advanced features example started', {
    port: PORT,
    environment: process.env.NODE_ENV || 'development',
    features: [
      'high-throughput-buffer',
      'global-error-capture',
      'health-checks',
      'performance-monitoring',
      'structured-logging'
    ],
    bufferStats: logger.getBufferStats()
  });
  
  console.log(`Advanced features example running on port ${PORT}`);
  console.log('Available endpoints:');
  console.log(`  GET  http://localhost:${PORT}/health`);
  console.log(`  GET  http://localhost:${PORT}/health/detailed`);
  console.log(`  GET  http://localhost:${PORT}/stats/buffer`);
  console.log(`  POST http://localhost:${PORT}/api/high-load`);
  console.log(`  GET  http://localhost:${PORT}/api/simulate/uncaught-exception`);
  console.log(`  GET  http://localhost:${PORT}/api/simulate/unhandled-rejection`);
  console.log(`  GET  http://localhost:${PORT}/api/simulate/memory-pressure`);
  console.log(`  POST http://localhost:${PORT}/api/user/:userId/action`);
  console.log(`  GET  http://localhost:${PORT}/api/performance`);
});

// Graceful shutdown
const gracefulShutdown = async (signal) => {
  logger.info('Graceful shutdown initiated', { 
    signal,
    bufferStats: logger.getBufferStats()
  });
  
  // Disable error capture
  errorHandler.disableGlobalErrorCapture();
  
  // Close logger
  await logger.close();
  
  console.log('Shutdown complete');
  process.exit(0);
};

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));