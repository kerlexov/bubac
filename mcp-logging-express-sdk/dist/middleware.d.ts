import { Request, Response, NextFunction } from 'express';
import { MiddlewareOptions } from './types';
export declare function createMiddleware(options: MiddlewareOptions): (req: Request, res: Response, next: NextFunction) => void;
//# sourceMappingURL=middleware.d.ts.map