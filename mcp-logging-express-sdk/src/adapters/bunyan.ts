import { MCPLogger } from '../logger';
import { MCPLoggerConfig } from '../types';

export class BunyanMCPStream {
  private mcpLogger: MCPLogger;
  public type = 'raw';

  constructor(config: MCPLoggerConfig) {
    this.mcpLogger = new MCPLogger(config);
  }

  write(record: any) {
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

  private mapBunyanLevel(bunyanLevel: number): string {
    // Bunyan levels: TRACE=10, DEBUG=20, INFO=30, WARN=40, ERROR=50, FATAL=60
    if (bunyanLevel <= 20) return 'DEBUG';
    if (bunyanLevel <= 30) return 'INFO';
    if (bunyanLevel <= 40) return 'WARN';
    if (bunyanLevel <= 50) return 'ERROR';
    return 'FATAL';
  }

  async close(): Promise<void> {
    await this.mcpLogger.close();
  }
}

// Helper function to create bunyan stream
export function createBunyanStream(config: MCPLoggerConfig) {
  return new BunyanMCPStream(config);
}