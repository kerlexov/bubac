import { MCPLoggerConfig } from '../types';
export declare class WinstonMCPTransport {
    private mcpLogger;
    name: string;
    constructor(config: MCPLoggerConfig);
    log(info: any, callback: () => void): void;
    private mapWinstonLevel;
    close(): Promise<void>;
}
export declare function createWinstonTransport(config: MCPLoggerConfig): WinstonMCPTransport;
//# sourceMappingURL=winston.d.ts.map