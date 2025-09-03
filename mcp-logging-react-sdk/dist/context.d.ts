import React, { ReactNode } from 'react';
import { MCPLogger, MCPLoggerConfig } from './types';
interface MCPLoggerProviderProps {
    config: MCPLoggerConfig;
    children: ReactNode;
}
export declare const MCPLoggerProvider: React.FC<MCPLoggerProviderProps>;
export declare const useMCPLogger: () => MCPLogger;
export declare const useMCPLoggerConfig: () => MCPLoggerConfig;
export {};
