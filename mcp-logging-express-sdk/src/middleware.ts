import { Request, Response, NextFunction } from 'express';
import { MCPLogger } from './logger';
import { MiddlewareOptions } from './types';
import * as os from 'os';
import * as url from 'url';

export function createMiddleware(options: MiddlewareOptions) {
  const logger = new MCPLogger(options);
  
  const {
    logRequests = true,
    logResponses = true,
    includeHeaders = false,
    includeBody = false,
    excludePaths = [],
    sensitiveHeaders = ['authorization', 'cookie', 'x-api-key']
  } = options;

  return (req: Request, res: Response, next: NextFunction) => {
    const startTime = Date.now();
    const requestId = req.headers['x-request-id'] || `req-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

    // Skip excluded paths
    if (excludePaths.some(path => req.path.startsWith(path))) {
      return next();
    }

    // Log incoming request
    if (logRequests) {
      const parsedUrl = url.parse(req.url, true);
      const requestMetadata: Record<string, any> = {
        requestId,
        method: req.method,
        url: req.url,
        path: req.path,
        query: req.query,
        ip: req.ip || req.connection.remoteAddress,
        userAgent: req.get('User-Agent'),
        protocol: req.protocol,
        hostname: req.hostname,
        originalUrl: req.originalUrl,
        baseUrl: req.baseUrl,
        fresh: req.fresh,
        stale: req.stale,
        xhr: req.xhr,
        secure: req.secure,
        httpVersion: req.httpVersion,
        contentType: req.get('Content-Type'),
        contentLength: req.get('Content-Length'),
        acceptLanguage: req.get('Accept-Language'),
        acceptEncoding: req.get('Accept-Encoding'),
        referer: req.get('Referer'),
        origin: req.get('Origin'),
        connection: {
          remoteAddress: req.connection.remoteAddress,
          remotePort: req.connection.remotePort,
          localAddress: req.connection.localAddress,
          localPort: req.connection.localPort
        },
        system: {
          hostname: os.hostname(),
          platform: os.platform(),
          arch: os.arch(),
          nodeVersion: process.version,
          pid: process.pid,
          uptime: process.uptime()
        }
      };

      if (includeHeaders) {
        requestMetadata.headers = sanitizeHeaders(req.headers, sensitiveHeaders);
      }

      if (includeBody && req.body) {
        requestMetadata.body = typeof req.body === 'object' ? 
          JSON.stringify(req.body).substring(0, 1000) : // Limit body size
          String(req.body).substring(0, 1000);
        requestMetadata.bodySize = JSON.stringify(req.body).length;
      }

      logger.info(`Incoming ${req.method} ${req.path}`, requestMetadata);
    }

    // Capture original res.end to log response
    const originalEnd = res.end;
    let responseLogged = false;

    res.end = function(chunk?: any, encoding?: any) {
      if (logResponses && !responseLogged) {
        responseLogged = true;
        const duration = Date.now() - startTime;
        
        const responseMetadata: Record<string, any> = {
          requestId,
          method: req.method,
          url: req.url,
          path: req.path,
          statusCode: res.statusCode,
          statusMessage: res.statusMessage,
          duration,
          contentLength: res.get('Content-Length'),
          contentType: res.get('Content-Type'),
          cacheControl: res.get('Cache-Control'),
          etag: res.get('ETag'),
          lastModified: res.get('Last-Modified'),
          location: res.get('Location'),
          setCookie: res.get('Set-Cookie'),
          performance: {
            responseTime: duration,
            memoryUsage: process.memoryUsage(),
            cpuUsage: process.cpuUsage ? process.cpuUsage() : undefined
          },
          response: {
            headersSent: res.headersSent,
            finished: res.finished,
            writableEnded: res.writableEnded
          }
        };

        if (includeHeaders) {
          responseMetadata.responseHeaders = res.getHeaders();
        }

        // Add response body if it's small and not binary
        if (chunk && typeof chunk === 'string' && chunk.length < 1000) {
          try {
            responseMetadata.responseBody = JSON.parse(chunk);
          } catch {
            responseMetadata.responseBody = chunk.substring(0, 500);
          }
        }

        const level = res.statusCode >= 500 ? 'error' : 
                     res.statusCode >= 400 ? 'warn' : 'info';
        
        logger[level](`${req.method} ${req.path} ${res.statusCode} - ${duration}ms`, responseMetadata);
      }

      return originalEnd.call(this, chunk, encoding);
    };

    // Handle errors
    const originalNext = next;
    next = (error?: any) => {
      if (error) {
        logger.error(`Request error: ${error.message}`, {
          requestId,
          method: req.method,
          url: req.url,
          path: req.path,
          error: {
            name: error.name,
            message: error.message,
            stack: error.stack
          }
        });
      }
      return originalNext(error);
    };

    next();
  };
}

function sanitizeHeaders(headers: Record<string, any>, sensitiveHeaders: string[]): Record<string, any> {
  const sanitized = { ...headers };
  
  sensitiveHeaders.forEach(header => {
    const lowerHeader = header.toLowerCase();
    Object.keys(sanitized).forEach(key => {
      if (key.toLowerCase() === lowerHeader) {
        sanitized[key] = '[REDACTED]';
      }
    });
  });

  return sanitized;
}