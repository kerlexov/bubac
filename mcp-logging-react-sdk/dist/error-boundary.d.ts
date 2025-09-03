import React, { Component, ErrorInfo, ReactNode } from 'react';
import { MCPLogger } from './types';
interface Props {
    logger: MCPLogger;
    fallback?: ReactNode;
    onError?: (error: Error, errorInfo: ErrorInfo) => void;
    children: ReactNode;
}
interface State {
    hasError: boolean;
    error?: Error;
}
export declare class MCPErrorBoundary extends Component<Props, State> {
    constructor(props: Props);
    static getDerivedStateFromError(error: Error): State;
    componentDidCatch(error: Error, errorInfo: ErrorInfo): void;
    render(): string | number | boolean | Iterable<React.ReactNode> | import("react/jsx-runtime").JSX.Element | null | undefined;
}
export declare function withMCPErrorBoundary<P extends object>(WrappedComponent: React.ComponentType<P>, logger: MCPLogger, fallback?: ReactNode): {
    (props: P): import("react/jsx-runtime").JSX.Element;
    displayName: string;
};
export {};
