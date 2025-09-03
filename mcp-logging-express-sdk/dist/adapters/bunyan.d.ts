import { MCPLoggerConfig } from '../types';
export declare class BunyanMCPStream {
    private mcpLogger;
    type: string;
    constructor(config: MCPLoggerConfig);
    write(record: any): void;
    private mapBunyanLevel;
    close(): Promise<void>;
}
export declare function createBunyanStream(config: MCPLoggerConfig): BunyanMCPStream;
//# sourceMappingURL=bunyan.d.ts.map