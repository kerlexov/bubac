const express = require('express');
const winston = require('winston');
const { createWinstonTransport } = require('../dist/index');

const app = express();
app.use(express.json());

// Create Winston logger with MCP transport
const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.errors({ stack: true }),
    winston.format.json()
  ),
  transports: [
    // Console transport for local development
    new winston.transports.Console({
      format: winston.format.combine(
        winston.format.colorize(),
        winston.format.simple()
      )
    }),
    // MCP transport for centralized logging
    createWinstonTransport({
      serverUrl: 'http://localhost:9080',
      serviceName: 'winston-example',
      agentId: 'winston-001'
    })
  ]
});

// Example routes using Winston logger
app.get('/api/data', (req, res) => {
  logger.info('Data requested', {
    query: req.query,
    userAgent: req.get('User-Agent'),
    ip: req.ip
  });
  
  const data = {
    items: ['item1', 'item2', 'item3'],
    timestamp: new Date().toISOString(),
    requestId: req.headers['x-request-id'] || 'unknown'
  };
  
  logger.info('Data response prepared', {
    itemCount: data.items.length,
    requestId: data.requestId
  });
  
  res.json(data);
});

app.post('/api/process', (req, res) => {
  const { action, payload } = req.body;
  
  logger.info('Processing request', {
    action,
    payloadSize: JSON.stringify(payload || {}).length
  });
  
  try {
    // Simulate processing
    if (action === 'error') {
      throw new Error('Simulated processing error');
    }
    
    const result = {
      success: true,
      action,
      processedAt: new Date().toISOString(),
      result: `Processed ${action} successfully`
    };
    
    logger.info('Request processed successfully', {
      action,
      processingTime: Math.random() * 100
    });
    
    res.json(result);
  } catch (error) {
    logger.error('Processing failed', {
      action,
      error: {
        name: error.name,
        message: error.message,
        stack: error.stack
      }
    });
    
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

// Demonstrate different log levels
app.get('/api/logs/test', (req, res) => {
  const testId = Date.now();
  
  logger.debug('Debug message', { testId, level: 'debug' });
  logger.info('Info message', { testId, level: 'info' });
  logger.warn('Warning message', { testId, level: 'warn' });
  logger.error('Error message', { testId, level: 'error' });
  
  res.json({
    message: 'Log test completed',
    testId,
    levels: ['debug', 'info', 'warn', 'error']
  });
});

const PORT = process.env.PORT || 3001;

app.listen(PORT, () => {
  logger.info('Winston integration example started', {
    port: PORT,
    environment: process.env.NODE_ENV || 'development',
    transports: logger.transports.length
  });
  
  console.log(`Winston integration example running on port ${PORT}`);
  console.log('Endpoints:');
  console.log(`  GET  http://localhost:${PORT}/api/data`);
  console.log(`  POST http://localhost:${PORT}/api/process`);
  console.log(`  GET  http://localhost:${PORT}/api/logs/test`);
});

// Graceful shutdown
const gracefulShutdown = async (signal) => {
  logger.info('Graceful shutdown initiated', { signal });
  
  // Close Winston transports
  logger.close();
  
  process.exit(0);
};

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));