"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createWinstonTransport = exports.WinstonMCPTransport = void 0;
const logger_1 = require("../logger");
class WinstonMCPTransport {
    constructor(config) {
        this.name = 'mcp-transport';
        this.mcpLogger = new logger_1.MCPLogger(config);
    }
    log(info, callback) {
        const { level, message, ...metadata } = info;
        // Map winston levels to MCP levels
        const mcpLevel = this.mapWinstonLevel(level);
        switch (mcpLevel) {
            case 'DEBUG':
                this.mcpLogger.debug(message, metadata);
                break;
            case 'INFO':
                this.mcpLogger.info(message, metadata);
                break;
            case 'WARN':
                this.mcpLogger.warn(message, metadata);
                break;
            case 'ERROR':
                this.mcpLogger.error(message, metadata);
                break;
            case 'FATAL':
                this.mcpLogger.fatal(message, metadata);
                break;
            default:
                this.mcpLogger.info(message, metadata);
        }
        callback();
    }
    mapWinstonLevel(winstonLevel) {
        const levelMap = {
            'silly': 'DEBUG',
            'debug': 'DEBUG',
            'verbose': 'DEBUG',
            'info': 'INFO',
            'warn': 'WARN',
            'error': 'ERROR',
            'crit': 'FATAL',
            'alert': 'FATAL',
            'emerg': 'FATAL'
        };
        return levelMap[winstonLevel] || 'INFO';
    }
    async close() {
        await this.mcpLogger.close();
    }
}
exports.WinstonMCPTransport = WinstonMCPTransport;
// Helper function to create winston transport
function createWinstonTransport(config) {
    return new WinstonMCPTransport(config);
}
exports.createWinstonTransport = createWinstonTransport;
//# sourceMappingURL=winston.js.map