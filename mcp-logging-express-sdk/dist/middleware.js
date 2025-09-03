"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createMiddleware = void 0;
const logger_1 = require("./logger");
const os = __importStar(require("os"));
const url = __importStar(require("url"));
function createMiddleware(options) {
    const logger = new logger_1.MCPLogger(options);
    const { logRequests = true, logResponses = true, includeHeaders = false, includeBody = false, excludePaths = [], sensitiveHeaders = ['authorization', 'cookie', 'x-api-key'] } = options;
    return (req, res, next) => {
        const startTime = Date.now();
        const requestId = req.headers['x-request-id'] || `req-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
        // Skip excluded paths
        if (excludePaths.some(path => req.path.startsWith(path))) {
            return next();
        }
        // Log incoming request
        if (logRequests) {
            const parsedUrl = url.parse(req.url, true);
            const requestMetadata = {
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
        res.end = function (chunk, encoding) {
            if (logResponses && !responseLogged) {
                responseLogged = true;
                const duration = Date.now() - startTime;
                const responseMetadata = {
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
                    }
                    catch (_a) {
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
        next = (error) => {
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
exports.createMiddleware = createMiddleware;
function sanitizeHeaders(headers, sensitiveHeaders) {
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
//# sourceMappingURL=middleware.js.map