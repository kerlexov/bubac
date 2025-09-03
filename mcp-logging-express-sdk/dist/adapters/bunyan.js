"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createBunyanStream = exports.BunyanMCPStream = void 0;
const logger_1 = require("../logger");
class BunyanMCPStream {
    constructor(config) {
        this.type = 'raw';
        this.mcpLogger = new logger_1.MCPLogger(config);
    }
    write(record) {
        const { level, msg, time, ...metadata } = record;
        // Map bunyan levels to MCP levels
        const mcpLevel = this.mapBunyanLevel(level);
        switch (mcpLevel) {
            case 'DEBUG':
                this.mcpLogger.debug(msg, metadata);
                break;
            case 'INFO':
                this.mcpLogger.info(msg, metadata);
                break;
            case 'WARN':
                this.mcpLogger.warn(msg, metadata);
                break;
            case 'ERROR':
                this.mcpLogger.error(msg, metadata);
                break;
            case 'FATAL':
                this.mcpLogger.fatal(msg, metadata);
                break;
            default:
                this.mcpLogger.info(msg, metadata);
        }
    }
    mapBunyanLevel(bunyanLevel) {
        // Bunyan levels: TRACE=10, DEBUG=20, INFO=30, WARN=40, ERROR=50, FATAL=60
        if (bunyanLevel <= 20)
            return 'DEBUG';
        if (bunyanLevel <= 30)
            return 'INFO';
        if (bunyanLevel <= 40)
            return 'WARN';
        if (bunyanLevel <= 50)
            return 'ERROR';
        return 'FATAL';
    }
    async close() {
        await this.mcpLogger.close();
    }
}
exports.BunyanMCPStream = BunyanMCPStream;
// Helper function to create bunyan stream
function createBunyanStream(config) {
    return new BunyanMCPStream(config);
}
exports.createBunyanStream = createBunyanStream;
//# sourceMappingURL=bunyan.js.map