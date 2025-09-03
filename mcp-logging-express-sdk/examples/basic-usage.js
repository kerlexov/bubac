const express = require('express');
const { createMiddleware, MCPLogger } = require('../dist/index');

const app = express();
app.use(express.json());

// Create a direct logger instance
const logger = new MCPLogger({
  serverUrl: 'http://localhost:8080',
  serviceName: 'example-api',
  agentId: 'api-001',
  bufferSize: 100,
  flushInterval: 2000
});

// Add MCP logging middleware
app.use(createMiddleware({
  serverUrl: 'http://localhost:8080',
  serviceName: 'example-api',
  agentId: 'api-001',
  logRequests: true,
  logResponses: true,
  includeHeaders: true,
  excludePaths: ['/health']
}));

// Health check endpoint (excluded from logging)
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Example API endpoints
app.get('/api/users', (req, res) => {
  logger.info('Fetching users', { 
    query: req.query,
    userAgent: req.get('User-Agent')
  });
  
  res.json({
    users: [
      { id: 1, name: 'John Doe', email: 'john@example.com' },
      { id: 2, name: 'Jane Smith', email: 'jane@example.com' }
    ]
  });
});

app.post('/api/users', (req, res) => {
  const userData = req.body;
  
  logger.info('Creating new user', {
    userData: { ...userData, password: '[REDACTED]' }
  });
  
  // Simulate validation error
  if (!userData.email) {
    logger.warn('User creation failed - missing email', { userData });
    return res.status(400).json({ error: 'Email is required' });
  }
  
  // Simulate successful creation
  const newUser = {
    id: Date.now(),
    ...userData,
    createdAt: new Date().toISOString()
  };
  
  logger.info('User created successfully', {
    userId: newUser.id,
    email: newUser.email
  });
  
  res.status(201).json(newUser);
});

app.get('/api/error', (req, res, next) => {
  const error = new Error('Simulated error for testing');
  error.code = 'TEST_ERROR';
  
  logger.error('Simulated error occurred', {
    error: {
      name: error.name,
      message: error.message,
      code: error.code,
      stack: error.stack
    },
    request: {
      method: req.method,
      url: req.url,
      ip: req.ip
    }
  });
  
  next(error);
});

// Error handling middleware
app.use((err, req, res, next) => {
  logger.error('Unhandled error', {
    error: {
      name: err.name,
      message: err.message,
      stack: err.stack
    },
    request: {
      method: req.method,
      url: req.url,
      headers: req.headers
    }
  });
  
  res.status(500).json({
    error: 'Internal server error',
    message: err.message
  });
});

const PORT = process.env.PORT || 3000;

app.listen(PORT, () => {
  logger.info('Server started', {
    port: PORT,
    environment: process.env.NODE_ENV || 'development',
    nodeVersion: process.version
  });
  
  console.log(`Server running on port ${PORT}`);
  console.log('Try these endpoints:');
  console.log(`  GET  http://localhost:${PORT}/health`);
  console.log(`  GET  http://localhost:${PORT}/api/users`);
  console.log(`  POST http://localhost:${PORT}/api/users`);
  console.log(`  GET  http://localhost:${PORT}/api/error`);
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  logger.info('Received SIGTERM, shutting down gracefully');
  await logger.close();
  process.exit(0);
});

process.on('SIGINT', async () => {
  logger.info('Received SIGINT, shutting down gracefully');
  await logger.close();
  process.exit(0);
});