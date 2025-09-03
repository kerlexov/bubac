import { MCPLogger } from './types';
export interface BrowserEnvironment {
    userAgent: string;
    platform: string;
    language: string;
    languages: string[];
    cookieEnabled: boolean;
    onLine: boolean;
    screenResolution: string;
    colorDepth: number;
    timezone: string;
    hardwareConcurrency: number;
    deviceMemory?: number;
    connection?: {
        effectiveType: string;
        downlink: number;
        rtt: number;
    };
}
export interface NavigationTiming {
    navigationStart: number;
    unloadEventStart: number;
    unloadEventEnd: number;
    redirectStart: number;
    redirectEnd: number;
    fetchStart: number;
    domainLookupStart: number;
    domainLookupEnd: number;
    connectStart: number;
    connectEnd: number;
    secureConnectionStart: number;
    requestStart: number;
    responseStart: number;
    responseEnd: number;
    domLoading: number;
    domInteractive: number;
    domContentLoadedEventStart: number;
    domContentLoadedEventEnd: number;
    domComplete: number;
    loadEventStart: number;
    loadEventEnd: number;
}
export declare class BrowserFeatures {
    private logger;
    private performanceObservers;
    private visibilityChangeHandler?;
    private beforeUnloadHandler?;
    private networkChangeHandler?;
    constructor(logger: MCPLogger);
    /**
     * Initialize all browser-specific features
     */
    initialize(): void;
    /**
     * Clean up all event listeners and observers
     */
    destroy(): void;
    /**
     * Get comprehensive browser environment information
     */
    getBrowserEnvironment(): BrowserEnvironment;
    /**
     * Get detailed navigation timing information
     */
    getNavigationTiming(): NavigationTiming | null;
    /**
     * Log browser environment information
     */
    private logBrowserEnvironment;
    /**
     * Setup comprehensive performance monitoring
     */
    private setupPerformanceMonitoring;
    /**
     * Capture detailed page load metrics
     */
    private capturePageLoadMetrics;
    /**
     * Setup Performance Observers for Core Web Vitals and other metrics
     */
    private setupPerformanceObservers;
    /**
     * Setup page visibility tracking
     */
    private setupVisibilityTracking;
    /**
     * Setup network connectivity monitoring
     */
    private setupNetworkMonitoring;
    /**
     * Setup page lifecycle tracking
     */
    private setupPageLifecycleTracking;
    /**
     * Enhanced user interaction tracking with more details
     */
    setupEnhancedUserInteractionTracking(): void;
    /**
     * Get a unique selector for an element
     */
    private getElementSelector;
}
