import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Component } from 'react';
export class MCPErrorBoundary extends Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false };
    }
    static getDerivedStateFromError(error) {
        return { hasError: true, error };
    }
    componentDidCatch(error, errorInfo) {
        const { logger, onError } = this.props;
        // Log the error through MCP Logger
        logger.error('React Error Boundary caught error', {
            error: error.message,
            stack: error.stack,
            componentStack: errorInfo.componentStack,
            errorBoundary: true,
        });
        // Call custom error handler if provided
        if (onError) {
            onError(error, errorInfo);
        }
    }
    render() {
        if (this.state.hasError) {
            // Render fallback UI
            if (this.props.fallback) {
                return this.props.fallback;
            }
            return (_jsxs("div", { style: {
                    padding: '20px',
                    border: '1px solid #ff6b6b',
                    borderRadius: '4px',
                    backgroundColor: '#ffe0e0',
                    color: '#d63031'
                }, children: [_jsx("h2", { children: "Something went wrong" }), _jsx("p", { children: "An error occurred in this component. The error has been logged." }), _jsxs("details", { style: { marginTop: '10px' }, children: [_jsx("summary", { children: "Error details" }), _jsx("pre", { style: {
                                    marginTop: '10px',
                                    padding: '10px',
                                    backgroundColor: '#f8f8f8',
                                    overflow: 'auto',
                                    fontSize: '12px'
                                }, children: this.state.error?.stack })] })] }));
        }
        return this.props.children;
    }
}
// HOC for wrapping components with error boundary
export function withMCPErrorBoundary(WrappedComponent, logger, fallback) {
    const WithErrorBoundary = (props) => (_jsx(MCPErrorBoundary, { logger: logger, fallback: fallback, children: _jsx(WrappedComponent, { ...props }) }));
    WithErrorBoundary.displayName = `withMCPErrorBoundary(${WrappedComponent.displayName || WrappedComponent.name})`;
    return WithErrorBoundary;
}
