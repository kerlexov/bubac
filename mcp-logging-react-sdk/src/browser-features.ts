import { MCPLogger, PerformanceMetrics, UserInteraction } from './types';

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

export class BrowserFeatures {
  private logger: MCPLogger;
  private performanceObservers: PerformanceObserver[] = [];
  private visibilityChangeHandler?: () => void;
  private beforeUnloadHandler?: () => void;
  private networkChangeHandler?: () => void;

  constructor(logger: MCPLogger) {
    this.logger = logger;
  }

  /**
   * Initialize all browser-specific features
   */
  initialize(): void {
    this.setupPerformanceMonitoring();
    this.setupVisibilityTracking();
    this.setupNetworkMonitoring();
    this.setupPageLifecycleTracking();
    this.logBrowserEnvironment();
  }

  /**
   * Clean up all event listeners and observers
   */
  destroy(): void {
    this.performanceObservers.forEach(observer => observer.disconnect());
    this.performanceObservers = [];

    if (this.visibilityChangeHandler) {
      document.removeEventListener('visibilitychange', this.visibilityChangeHandler);
    }

    if (this.beforeUnloadHandler) {
      window.removeEventListener('beforeunload', this.beforeUnloadHandler);
    }

    if (this.networkChangeHandler) {
      window.removeEventListener('online', this.networkChangeHandler);
      window.removeEventListener('offline', this.networkChangeHandler);
    }
  }

  /**
   * Get comprehensive browser environment information
   */
  getBrowserEnvironment(): BrowserEnvironment {
    const nav = navigator as any;
    
    return {
      userAgent: navigator.userAgent,
      platform: navigator.platform,
      language: navigator.language,
      languages: navigator.languages ? Array.from(navigator.languages) : [navigator.language],
      cookieEnabled: navigator.cookieEnabled,
      onLine: navigator.onLine,
      screenResolution: `${screen.width}x${screen.height}`,
      colorDepth: screen.colorDepth,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      hardwareConcurrency: navigator.hardwareConcurrency || 0,
      deviceMemory: nav.deviceMemory,
      connection: nav.connection ? {
        effectiveType: nav.connection.effectiveType,
        downlink: nav.connection.downlink,
        rtt: nav.connection.rtt,
      } : undefined,
    };
  }

  /**
   * Get detailed navigation timing information
   */
  getNavigationTiming(): NavigationTiming | null {
    if (!performance.timing) {
      return null;
    }

    const timing = performance.timing;
    return {
      navigationStart: timing.navigationStart,
      unloadEventStart: timing.unloadEventStart,
      unloadEventEnd: timing.unloadEventEnd,
      redirectStart: timing.redirectStart,
      redirectEnd: timing.redirectEnd,
      fetchStart: timing.fetchStart,
      domainLookupStart: timing.domainLookupStart,
      domainLookupEnd: timing.domainLookupEnd,
      connectStart: timing.connectStart,
      connectEnd: timing.connectEnd,
      secureConnectionStart: timing.secureConnectionStart,
      requestStart: timing.requestStart,
      responseStart: timing.responseStart,
      responseEnd: timing.responseEnd,
      domLoading: timing.domLoading,
      domInteractive: timing.domInteractive,
      domContentLoadedEventStart: timing.domContentLoadedEventStart,
      domContentLoadedEventEnd: timing.domContentLoadedEventEnd,
      domComplete: timing.domComplete,
      loadEventStart: timing.loadEventStart,
      loadEventEnd: timing.loadEventEnd,
    };
  }

  /**
   * Log browser environment information
   */
  private logBrowserEnvironment(): void {
    const env = this.getBrowserEnvironment();
    this.logger.info('Browser environment detected', {
      type: 'browser_environment',
      environment: env,
    });
  }

  /**
   * Setup comprehensive performance monitoring
   */
  private setupPerformanceMonitoring(): void {
    if (typeof window === 'undefined' || !('performance' in window)) {
      return;
    }

    // Monitor page load performance
    window.addEventListener('load', () => {
      setTimeout(() => {
        this.capturePageLoadMetrics();
      }, 0);
    });

    // Setup Performance Observer for various metrics
    this.setupPerformanceObservers();
  }

  /**
   * Capture detailed page load metrics
   */
  private capturePageLoadMetrics(): void {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
    const paint = performance.getEntriesByType('paint');
    const timing = this.getNavigationTiming();

    if (navigation) {
      const metrics: PerformanceMetrics = {
        pageLoadTime: navigation.loadEventEnd - navigation.fetchStart,
        domContentLoadedTime: navigation.domContentLoadedEventEnd - navigation.fetchStart,
        firstContentfulPaint: paint.find(p => p.name === 'first-contentful-paint')?.startTime,
      };

      // Add detailed timing breakdown
      const detailedMetrics = {
        ...metrics,
        dnsLookupTime: navigation.domainLookupEnd - navigation.domainLookupStart,
        tcpConnectTime: navigation.connectEnd - navigation.connectStart,
        tlsTime: navigation.connectEnd - navigation.secureConnectionStart,
        requestTime: navigation.responseStart - navigation.requestStart,
        responseTime: navigation.responseEnd - navigation.responseStart,
        domProcessingTime: navigation.domComplete - navigation.domInteractive,
        navigationTiming: timing,
      };

      this.logger.logPerformance(detailedMetrics);
      this.logger.info('Page load metrics captured', {
        type: 'page_load_complete',
        metrics: detailedMetrics,
      });
    }
  }

  /**
   * Setup Performance Observers for Core Web Vitals and other metrics
   */
  private setupPerformanceObservers(): void {
    if (!('PerformanceObserver' in window)) {
      return;
    }

    try {
      // Largest Contentful Paint (LCP)
      const lcpObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        const lastEntry = entries[entries.length - 1];
        
        this.logger.logPerformance({
          largestContentfulPaint: lastEntry.startTime,
        });

        this.logger.info('LCP measured', {
          type: 'core_web_vital',
          metric: 'lcp',
          value: lastEntry.startTime,
          element: (lastEntry as any).element?.tagName,
        });
      });
      lcpObserver.observe({ entryTypes: ['largest-contentful-paint'] });
      this.performanceObservers.push(lcpObserver);

      // Cumulative Layout Shift (CLS)
      let clsValue = 0;
      const clsObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (!(entry as any).hadRecentInput) {
            clsValue += (entry as any).value;
          }
        }
        
        if (clsValue > 0) {
          this.logger.logPerformance({
            cumulativeLayoutShift: clsValue,
          });

          this.logger.info('CLS measured', {
            type: 'core_web_vital',
            metric: 'cls',
            value: clsValue,
          });
        }
      });
      clsObserver.observe({ entryTypes: ['layout-shift'] });
      this.performanceObservers.push(clsObserver);

      // First Input Delay (FID)
      const fidObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          const fid = (entry as any).processingStart - entry.startTime;
          
          this.logger.logPerformance({
            firstInputDelay: fid,
          });

          this.logger.info('FID measured', {
            type: 'core_web_vital',
            metric: 'fid',
            value: fid,
            eventType: (entry as any).name,
          });
        }
      });
      fidObserver.observe({ entryTypes: ['first-input'] });
      this.performanceObservers.push(fidObserver);

      // Long Tasks (performance bottlenecks)
      const longTaskObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          this.logger.warn('Long task detected', {
            type: 'performance_issue',
            duration: entry.duration,
            startTime: entry.startTime,
            name: entry.name,
          });
        }
      });
      longTaskObserver.observe({ entryTypes: ['longtask'] });
      this.performanceObservers.push(longTaskObserver);

      // Resource timing
      const resourceObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          const resource = entry as PerformanceResourceTiming;
          
          // Log slow resources
          if (resource.duration > 1000) {
            this.logger.warn('Slow resource detected', {
              type: 'slow_resource',
              name: resource.name,
              duration: resource.duration,
              size: resource.transferSize,
              type_detail: resource.initiatorType,
            });
          }
        }
      });
      resourceObserver.observe({ entryTypes: ['resource'] });
      this.performanceObservers.push(resourceObserver);

    } catch (error) {
      this.logger.warn('Failed to setup performance observers', {
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }

  /**
   * Setup page visibility tracking
   */
  private setupVisibilityTracking(): void {
    if (typeof document === 'undefined') {
      return;
    }

    this.visibilityChangeHandler = () => {
      const isVisible = !document.hidden;
      
      this.logger.logUserInteraction({
        type: 'custom',
        timestamp: new Date(),
        metadata: {
          action: 'visibility_change',
          visible: isVisible,
          visibilityState: document.visibilityState,
        },
      });

      this.logger.info(`Page ${isVisible ? 'visible' : 'hidden'}`, {
        type: 'page_visibility',
        visible: isVisible,
        visibilityState: document.visibilityState,
      });
    };

    document.addEventListener('visibilitychange', this.visibilityChangeHandler);
  }

  /**
   * Setup network connectivity monitoring
   */
  private setupNetworkMonitoring(): void {
    if (typeof window === 'undefined') {
      return;
    }

    this.networkChangeHandler = () => {
      const isOnline = navigator.onLine;
      
      this.logger.info(`Network ${isOnline ? 'connected' : 'disconnected'}`, {
        type: 'network_change',
        online: isOnline,
        timestamp: new Date().toISOString(),
      });

      this.logger.logUserInteraction({
        type: 'custom',
        timestamp: new Date(),
        metadata: {
          action: 'network_change',
          online: isOnline,
        },
      });
    };

    window.addEventListener('online', this.networkChangeHandler);
    window.addEventListener('offline', this.networkChangeHandler);

    // Monitor connection changes if supported
    const connection = (navigator as any).connection;
    if (connection) {
      const connectionChangeHandler = () => {
        this.logger.info('Connection changed', {
          type: 'connection_change',
          effectiveType: connection.effectiveType,
          downlink: connection.downlink,
          rtt: connection.rtt,
        });
      };

      connection.addEventListener('change', connectionChangeHandler);
    }
  }

  /**
   * Setup page lifecycle tracking
   */
  private setupPageLifecycleTracking(): void {
    if (typeof window === 'undefined') {
      return;
    }

    // Track page unload
    this.beforeUnloadHandler = () => {
      this.logger.info('Page unloading', {
        type: 'page_lifecycle',
        action: 'beforeunload',
        timestamp: new Date().toISOString(),
      });

      // Force flush logs before page unloads
      this.logger.flush().catch(() => {
        // Ignore flush errors on unload
      });
    };

    window.addEventListener('beforeunload', this.beforeUnloadHandler);

    // Track focus/blur events
    window.addEventListener('focus', () => {
      this.logger.logUserInteraction({
        type: 'custom',
        timestamp: new Date(),
        metadata: { action: 'window_focus' },
      });
    });

    window.addEventListener('blur', () => {
      this.logger.logUserInteraction({
        type: 'custom',
        timestamp: new Date(),
        metadata: { action: 'window_blur' },
      });
    });

    // Track resize events (throttled)
    let resizeTimeout: NodeJS.Timeout;
    window.addEventListener('resize', () => {
      clearTimeout(resizeTimeout);
      resizeTimeout = setTimeout(() => {
        this.logger.logUserInteraction({
          type: 'custom',
          timestamp: new Date(),
          metadata: {
            action: 'window_resize',
            width: window.innerWidth,
            height: window.innerHeight,
          },
        });
      }, 250);
    });
  }

  /**
   * Enhanced user interaction tracking with more details
   */
  setupEnhancedUserInteractionTracking(): void {
    if (typeof document === 'undefined') {
      return;
    }

    // Enhanced click tracking
    document.addEventListener('click', (event) => {
      const target = event.target as HTMLElement;
      const element = this.getElementSelector(target);
      
      this.logger.logUserInteraction({
        type: 'click',
        element,
        timestamp: new Date(),
        metadata: {
          x: event.clientX,
          y: event.clientY,
          button: event.button,
          ctrlKey: event.ctrlKey,
          shiftKey: event.shiftKey,
          altKey: event.altKey,
          metaKey: event.metaKey,
          tagName: target.tagName,
          className: target.className,
          id: target.id,
          text: target.textContent?.substring(0, 100),
        },
      });
    });

    // Form interaction tracking
    document.addEventListener('submit', (event) => {
      const form = event.target as HTMLFormElement;
      const formData = new FormData(form);
      const fields = Array.from(formData.keys());
      
      this.logger.logUserInteraction({
        type: 'custom',
        timestamp: new Date(),
        metadata: {
          action: 'form_submit',
          formId: form.id,
          formName: form.name,
          fieldCount: fields.length,
          fields: fields,
        },
      });
    });

    // Input focus tracking
    document.addEventListener('focusin', (event) => {
      const target = event.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') {
        this.logger.logUserInteraction({
          type: 'input',
          element: this.getElementSelector(target),
          timestamp: new Date(),
          metadata: {
            action: 'focus',
            inputType: (target as HTMLInputElement).type,
            name: (target as HTMLInputElement).name,
          },
        });
      }
    });

    // Scroll tracking with more details
    let scrollTimeout: NodeJS.Timeout;
    let lastScrollY = window.scrollY;
    
    document.addEventListener('scroll', () => {
      clearTimeout(scrollTimeout);
      scrollTimeout = setTimeout(() => {
        const currentScrollY = window.scrollY;
        const direction = currentScrollY > lastScrollY ? 'down' : 'up';
        const scrollPercent = Math.round((currentScrollY / (document.body.scrollHeight - window.innerHeight)) * 100);
        
        this.logger.logUserInteraction({
          type: 'scroll',
          timestamp: new Date(),
          metadata: {
            scrollY: currentScrollY,
            scrollX: window.scrollX,
            direction,
            scrollPercent: Math.min(100, Math.max(0, scrollPercent)),
            pageHeight: document.body.scrollHeight,
            viewportHeight: window.innerHeight,
          },
        });
        
        lastScrollY = currentScrollY;
      }, 250);
    });
  }

  /**
   * Get a unique selector for an element
   */
  private getElementSelector(element: HTMLElement): string {
    const parts: string[] = [];
    
    let current: HTMLElement | null = element;
    while (current && current !== document.body) {
      let selector = current.tagName.toLowerCase();
      
      if (current.id) {
        selector += `#${current.id}`;
        parts.unshift(selector);
        break;
      }
      
      if (current.className) {
        const classes = current.className.split(' ').filter(c => c.trim());
        if (classes.length > 0) {
          selector += `.${classes.join('.')}`;
        }
      }
      
      // Add nth-child if needed for uniqueness
      const siblings = Array.from(current.parentElement?.children || []);
      const sameTagSiblings = siblings.filter(s => s.tagName === current!.tagName);
      if (sameTagSiblings.length > 1) {
        const index = sameTagSiblings.indexOf(current) + 1;
        selector += `:nth-child(${index})`;
      }
      
      parts.unshift(selector);
      current = current.parentElement;
    }
    
    return parts.join(' > ');
  }
}