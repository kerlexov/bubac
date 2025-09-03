import express, { Request, Response } from 'express';
import request from 'supertest';
import { createMiddleware } from '../middleware';

// Mock the logger
jest.mock('../logger');

describe('Express Middleware', () => {
  let app: express.Application;
  
  const mockConfig = {
    serverUrl: 'http://localhost:9080',
    serviceName: 'test-service',
    agentId: 'test-agent'
  };

  beforeEach(() => {
    app = express();
    app.use(express.json());
  });

  test('should log incoming requests', async () => {
    app.use(createMiddleware(mockConfig));
    app.get('/test', (req: Request, res: Response) => {
      res.json({ message: 'success' });
    });

    const response = await request(app)
      .get('/test')
      .expect(200);

    expect(response.body).toEqual({ message: 'success' });
  });

  test('should log request with query parameters', async () => {
    app.use(createMiddleware(mockConfig));
    app.get('/test', (req: Request, res: Response) => {
      res.json({ query: req.query });
    });

    await request(app)
      .get('/test?param1=value1&param2=value2')
      .expect(200);
  });

  test('should log POST requests with body', async () => {
    app.use(createMiddleware({
      ...mockConfig,
      includeBody: true
    }));
    
    app.post('/test', (req: Request, res: Response) => {
      res.json({ received: req.body });
    });

    await request(app)
      .post('/test')
      .send({ test: 'data' })
      .expect(200);
  });

  test('should exclude specified paths', async () => {
    app.use(createMiddleware({
      ...mockConfig,
      excludePaths: ['/health']
    }));
    
    app.get('/health', (req: Request, res: Response) => {
      res.json({ status: 'ok' });
    });

    await request(app)
      .get('/health')
      .expect(200);
  });

  test('should sanitize sensitive headers', async () => {
    app.use(createMiddleware({
      ...mockConfig,
      includeHeaders: true,
      sensitiveHeaders: ['authorization']
    }));
    
    app.get('/test', (req: Request, res: Response) => {
      res.json({ message: 'success' });
    });

    await request(app)
      .get('/test')
      .set('Authorization', 'Bearer secret-token')
      .expect(200);
  });

  test('should log errors', async () => {
    app.use(createMiddleware(mockConfig));
    
    app.get('/error', (req: Request, res: Response, next) => {
      const error = new Error('Test error');
      next(error);
    });

    app.use((err: Error, req: Request, res: Response, next: any) => {
      res.status(500).json({ error: err.message });
    });

    await request(app)
      .get('/error')
      .expect(500);
  });

  test('should log different status codes appropriately', async () => {
    app.use(createMiddleware(mockConfig));
    
    app.get('/success', (req: Request, res: Response) => {
      res.status(200).json({ status: 'ok' });
    });
    
    app.get('/not-found', (req: Request, res: Response) => {
      res.status(404).json({ error: 'Not found' });
    });
    
    app.get('/server-error', (req: Request, res: Response) => {
      res.status(500).json({ error: 'Server error' });
    });

    await request(app).get('/success').expect(200);
    await request(app).get('/not-found').expect(404);
    await request(app).get('/server-error').expect(500);
  });
});