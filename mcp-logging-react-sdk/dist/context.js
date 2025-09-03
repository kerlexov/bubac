import { jsx as _jsx } from "react/jsx-runtime";
import { createContext, useContext, useEffect, useRef } from 'react';
import { MCPLoggerImpl } from './logger';
const MCPLoggerContext = createContext(null);
export const MCPLoggerProvider = ({ config, children }) => {
    const loggerRef = useRef(null);
    // Initialize logger only once
    if (!loggerRef.current) {
        loggerRef.current = new MCPLoggerImpl(config);
    }
    useEffect(() => {
        const logger = loggerRef.current;
        // Log provider initialization
        logger.info('MCPLoggerProvider initialized', {
            serviceName: config.serviceName,
            agentId: config.agentId,
        });
        // Setup performance monitoring if enabled
        if (config.enablePerformanceMetrics !== false) {
            setupPerformanceMonitoring(logger);
        }
        // Setup user interaction monitoring if enabled
        if (config.enableUserInteractions !== false) {
            setupUserInteractionMonitoring(logger);
        }
        // Cleanup on unmount
        return () => {
            logger.destroy();
        };
    }, [config]);
    const contextValue = {
        logger: loggerRef.current,
        config,
    };
    return (_jsx(MCPLoggerContext.Provider, { value: contextValue, children: children }));
};
export const useMCPLogger = () => {
    const context = useContext(MCPLoggerContext);
    if (!context) {
        throw new Error('useMCPLogger must be used within an MCPLoggerProvider');
    }
    return context.logger;
};
export const useMCPLoggerConfig = () => {
    const context = useContext(MCPLoggerContext);
    if (!context) {
        throw new Error('useMCPLoggerConfig must be used within an MCPLoggerProvider');
    }
    return context.config;
};
function setupPerformanceMonitoring(logger) {
    // Monitor page load performance
    if (typeof window !== 'undefined' && 'performance' in window) {
        window.addEventListener('load', () => {
            setTimeout(() => {
                const navigation = performance.getEntriesByType('navigation')[0];
                const paint = performance.getEntriesByType('paint');
                const metrics = {
                    pageLoadTime: navigation.loadEventEnd - navigation.fetchStart,
                    domContentLoadedTime: navigation.domContentLoadedEventEnd - navigation.fetchStart,
                    firstContentfulPaint: paint.find(p => p.name === 'first-contentful-paint')?.startTime,
                };
                logger.logPerformance(metrics);
            }, 0);
        });
        // Monitor Core Web Vitals if available
        if ('PerformanceObserver' in window) {
            try {
                // Largest Contentful Paint
                const lcpObserver = new PerformanceObserver((list) => {
                    const entries = list.getEntries();
                    const lastEntry = entries[entries.length - 1];
                    logger.logPerformance({
                        largestContentfulPaint: lastEntry.startTime,
                    });
                });
                lcpObserver.observe({ entryTypes: ['largest-contentful-paint'] });
                // Cumulative Layout Shift
                const clsObserver = new PerformanceObserver((list) => {
                    let clsValue = 0;
                    for (const entry of list.getEntries()) {
                        if (!entry.hadRecentInput) {
                            clsValue += entry.value;
                        }
                    }
                    if (clsValue > 0) {
                        logger.logPerformance({
                            cumulativeLayoutShift: clsValue,
                        });
                    }
                });
                clsObserver.observe({ entryTypes: ['layout-shift'] });
                // First Input Delay
                const fidObserver = new PerformanceObserver((list) => {
                    for (const entry of list.getEntries()) {
                        logger.logPerformance({
                            firstInputDelay: entry.processingStart - entry.startTime,
                        });
                    }
                });
                fidObserver.observe({ entryTypes: ['first-input'] });
            }
            catch (error) {
                // PerformanceObserver not fully supported, ignore
            }
        }
    }
}
function setupUserInteractionMonitoring(logger) {
    if (typeof window === 'undefined') {
        return;
    }
    // Track clicks
    document.addEventListener('click', (event) => {
        const target = event.target;
        const element = target.tagName.toLowerCase() +
            (target.id ? `#${target.id}` : '') +
            (target.className ? `.${target.className.split(' ').join('.')}` : '');
        logger.logUserInteraction({
            type: 'click',
            element,
            timestamp: new Date(),
            metadata: {
                x: event.clientX,
                y: event.clientY,
                button: event.button,
            },
        });
    });
    // Track navigation
    let currentUrl = window.location.href;
    const checkUrlChange = () => {
        if (window.location.href !== currentUrl) {
            logger.logUserInteraction({
                type: 'navigation',
                url: window.location.href,
                timestamp: new Date(),
                metadata: {
                    from: currentUrl,
                    to: window.location.href,
                },
            });
            currentUrl = window.location.href;
        }
    };
    // Check for URL changes (for SPAs)
    setInterval(checkUrlChange, 1000);
    // Track scroll events (throttled)
    let scrollTimeout;
    document.addEventListener('scroll', () => {
        clearTimeout(scrollTimeout);
        scrollTimeout = setTimeout(() => {
            logger.logUserInteraction({
                type: 'scroll',
                timestamp: new Date(),
                metadata: {
                    scrollY: window.scrollY,
                    scrollX: window.scrollX,
                },
            });
        }, 250);
    });
}
