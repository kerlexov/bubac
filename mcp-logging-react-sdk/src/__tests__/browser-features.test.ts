import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { BrowserFeatures } from '../browser-features';
import { MCPLogger } from '../types';

describe('BrowserFeatures', () => {
  let mockLogger: MCPLogger;
  let browserFeatures: BrowserFeatures;

  beforeEach(() => {
    mockLogger = {
      debug: vi.fn(),
      info: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
      fatal: vi.fn(),
      logPerformance: vi.fn(),
      logUserInteraction: vi.fn(),
      flush: vi.fn(),
      getHealthStatus: vi.fn(() => ({ isHealthy: true })),
      getStorageStats: vi.fn(() => null),
    };

    // Mock browser APIs
    global.performance = {
      getEntriesByType: vi.fn(() => []),
      timing: {
        navigationStart: 1000,
        loadEventEnd: 2000,
        fetchStart: 1100,
        domContentLoadedEventEnd: 1800,
        domainLookupStart: 1200,
        domainLookupEnd: 1250,
        connectStart: 1250,
        connectEnd: 1300,
        secureConnectionStart: 1275,
        requestStart: 1300,
        responseStart: 1400,
        responseEnd: 1500,
        domLoading: 1500,
        domComplete: 1900,
      },
    } as any;

    Object.defineProperty(window, 'navigator', {
      value: {
        userAgent: 'Test Browser',
        platform: 'Test Platform',
        language: 'en-US',
        languages: ['en-US', 'en'],
        cookieEnabled: true,
        onLine: true,
        hardwareConcurrency: 4,
      },
      writable: true,
    });

    Object.defineProperty(window, 'screen', {
      value: {
        width: 1920,
        height: 1080,
        colorDepth: 24,
      },
      writable: true,
    });

    browserFeatures = new BrowserFeatures(mockLogger);
  });

  afterEach(() => {
    browserFeatures.destroy();
  });

  describe('Browser Environment Detection', () => {
    it('should detect browser environment correctly', () => {
      const env = browserFeatures.getBrowserEnvironment();

      expect(env).toEqual({
        userAgent: 'Test Browser',
        platform: 'Test Platform',
        language: 'en-US',
        languages: ['en-US', 'en'],
        cookieEnabled: true,
        onLine: true,
        screenResolution: '1920x1080',
        colorDepth: 24,
        timezone: expect.any(String),
        hardwareConcurrency: 4,
        deviceMemory: undefined,
        connection: undefined,
      });
    });

    it('should log browser environment on initialization', () => {
      browserFeatures.initialize();

      expect(mockLogger.info).toHaveBeenCalledWith(
        'Browser environment detected',
        expect.objectContaining({
          type: 'browser_environment',
          environment: expect.any(Object),
        })
      );
    });
  });

  describe('Navigation Timing', () => {
    it('should get navigation timing information', () => {
      const timing = browserFeatures.getNavigationTiming();

      expect(timing).toEqual({
        navigationStart: 1000,
        unloadEventStart: undefined,
        unloadEventEnd: undefined,
        redirectStart: undefined,
        redirectEnd: undefined,
        fetchStart: 1100,
        domainLookupStart: 1200,
        domainLookupEnd: 1250,
        connectStart: 1250,
        connectEnd: 1300,
        secureConnectionStart: 1275,
        requestStart: 1300,
        responseStart: 1400,
        responseEnd: 1500,
        domLoading: 1500,
        domInteractive: undefined,
        domContentLoadedEventStart: undefined,
        domContentLoadedEventEnd: 1800,
        domComplete: 1900,
        loadEventStart: undefined,
        loadEventEnd: 2000,
      });
    });

    it('should return null when performance.timing is not available', () => {
      global.performance = {} as any;
      const timing = browserFeatures.getNavigationTiming();
      expect(timing).toBeNull();
    });
  });

  describe('Performance Monitoring', () => {
    it('should setup performance observers when available', () => {
      const mockObserver = {
        observe: vi.fn(),
        disconnect: vi.fn(),
      };

      global.PerformanceObserver = vi.fn().mockImplementation(() => mockObserver);

      browserFeatures.initialize();

      expect(global.PerformanceObserver).toHaveBeenCalledTimes(5); // LCP, CLS, FID, longtask, resource
      expect(mockObserver.observe).toHaveBeenCalledTimes(5);
    });

    it('should handle PerformanceObserver errors gracefully', () => {
      global.PerformanceObserver = vi.fn().mockImplementation(() => {
        throw new Error('PerformanceObserver not supported');
      });

      expect(() => browserFeatures.initialize()).not.toThrow();
      expect(mockLogger.warn).toHaveBeenCalledWith(
        'Failed to setup performance observers',
        expect.objectContaining({
          error: 'PerformanceObserver not supported',
        })
      );
    });
  });

  describe('Visibility Tracking', () => {
    it('should setup visibility change listener', () => {
      const addEventListenerSpy = vi.spyOn(document, 'addEventListener');
      
      browserFeatures.initialize();

      expect(addEventListenerSpy).toHaveBeenCalledWith(
        'visibilitychange',
        expect.any(Function)
      );
    });

    it('should log visibility changes', () => {
      Object.defineProperty(document, 'hidden', {
        value: false,
        writable: true,
      });

      Object.defineProperty(document, 'visibilityState', {
        value: 'visible',
        writable: true,
      });

      browserFeatures.initialize();

      // Simulate visibility change
      const visibilityHandler = document.addEventListener.mock.calls
        .find(call => call[0] === 'visibilitychange')?.[1] as Function;

      if (visibilityHandler) {
        visibilityHandler();

        expect(mockLogger.info).toHaveBeenCalledWith(
          'Page visible',
          expect.objectContaining({
            type: 'page_visibility',
            visible: true,
            visibilityState: 'visible',
          })
        );
      }
    });
  });

  describe('Network Monitoring', () => {
    it('should setup network change listeners', () => {
      const addEventListenerSpy = vi.spyOn(window, 'addEventListener');
      
      browserFeatures.initialize();

      expect(addEventListenerSpy).toHaveBeenCalledWith('online', expect.any(Function));
      expect(addEventListenerSpy).toHaveBeenCalledWith('offline', expect.any(Function));
    });

    it('should log network status changes', () => {
      Object.defineProperty(navigator, 'onLine', {
        value: true,
        writable: true,
      });

      browserFeatures.initialize();

      // Simulate network change
      const onlineHandler = window.addEventListener.mock.calls
        .find(call => call[0] === 'online')?.[1] as Function;

      if (onlineHandler) {
        onlineHandler();

        expect(mockLogger.info).toHaveBeenCalledWith(
          'Network connected',
          expect.objectContaining({
            type: 'network_change',
            online: true,
          })
        );
      }
    });
  });

  describe('Enhanced User Interaction Tracking', () => {
    it('should setup enhanced click tracking', () => {
      const addEventListenerSpy = vi.spyOn(document, 'addEventListener');
      
      browserFeatures.setupEnhancedUserInteractionTracking();

      expect(addEventListenerSpy).toHaveBeenCalledWith('click', expect.any(Function));
      expect(addEventListenerSpy).toHaveBeenCalledWith('submit', expect.any(Function));
      expect(addEventListenerSpy).toHaveBeenCalledWith('focusin', expect.any(Function));
      expect(addEventListenerSpy).toHaveBeenCalledWith('scroll', expect.any(Function));
    });

    it('should track click events with detailed metadata', () => {
      browserFeatures.setupEnhancedUserInteractionTracking();

      const clickHandler = document.addEventListener.mock.calls
        .find(call => call[0] === 'click')?.[1] as Function;

      if (clickHandler) {
        const mockEvent = {
          target: {
            tagName: 'BUTTON',
            id: 'test-button',
            className: 'btn primary',
            textContent: 'Click me',
          },
          clientX: 100,
          clientY: 200,
          button: 0,
          ctrlKey: false,
          shiftKey: false,
          altKey: false,
          metaKey: false,
        };

        clickHandler(mockEvent);

        expect(mockLogger.logUserInteraction).toHaveBeenCalledWith({
          type: 'click',
          element: expect.any(String),
          timestamp: expect.any(Date),
          metadata: expect.objectContaining({
            x: 100,
            y: 200,
            button: 0,
            tagName: 'BUTTON',
            className: 'btn primary',
            id: 'test-button',
            text: 'Click me',
          }),
        });
      }
    });
  });

  describe('Cleanup', () => {
    it('should cleanup all listeners and observers on destroy', () => {
      const mockObserver = {
        observe: vi.fn(),
        disconnect: vi.fn(),
      };

      global.PerformanceObserver = vi.fn().mockImplementation(() => mockObserver);

      browserFeatures.initialize();
      browserFeatures.setupEnhancedUserInteractionTracking();

      browserFeatures.destroy();

      expect(mockObserver.disconnect).toHaveBeenCalled();
    });
  });
});