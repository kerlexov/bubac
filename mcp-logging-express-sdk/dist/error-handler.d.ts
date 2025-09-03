import { MCPLogger } from './logger';
export declare class ErrorHandler {
    private logger;
    private originalUncaughtException?;
    private originalUnhandledRejection?;
    private originalWarning?;
    constructor(logger: MCPLogger);
    enableGlobalErrorCapture(): void;
    disableGlobalErrorCapture(): void;
}
//# sourceMappingURL=error-handler.d.ts.map