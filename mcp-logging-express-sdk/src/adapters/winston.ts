import { MCPLogger } from '../logger';
import { MCPLoggerConfig } from '../types';

export class WinstonMCPTransport {
  private mcpLogger: MCPLogger;
  public name = 'mcp-transport';

  constructor(config: MCPLoggerConfig) {
    this.mcpLogger = new MCPLogger(config);
  }

  log(info: any, callback: () => void) {
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

  private mapWinstonLevel(winstonLevel: string): string {
    const levelMap: Record<string, string> = {
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

  async close(): Promise<void> {
    await this.mcpLogger.close();
  }
}

// Helper function to create winston transport
export function createWinstonTransport(config: MCPLoggerConfig) {
  return new WinstonMCPTransport(config);
}