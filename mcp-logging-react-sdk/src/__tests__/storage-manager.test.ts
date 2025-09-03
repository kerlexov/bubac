import { describe, it, expect, vi, beforeEach } from 'vitest';
import { LocalStorageManager } from '../storage-manager';
import { LogEntry } from '../types';

describe('LocalStorageManager', () => {
  let storageManager: LocalStorageManager;
  let mockLogs: LogEntry[];

  beforeEach(() => {
    // Clear localStorage
    localStorage.clear();

    storageManager = new LocalStorageManager({
      keyPrefix: 'test-logger',
      maxEntries: 5,
      maxSizeBytes: 1024, // 1KB for testing
    });

    mockLogs = [
      {
        id: '1',
        timestamp: new Date('2024-01-01T10:00:00Z'),
        level: 'INFO',
        message: 'Test message 1',
        serviceName: 'test-service',
        agentId: 'test-agent',
        platform: 'react',
      },
      {
        id: '2',
        timestamp: new Date('2024-01-01T10:01:00Z'),
        level: 'ERROR',
        message: 'Test error message',
        serviceName: 'test-service',
        agentId: 'test-agent',
        platform: 'react',
      },
    ];
  });

  describe('Basic Operations', () => {
    it('should save and load logs', () => {
      const success = storageManager.saveLogs(mockLogs);
      expect(success).toBe(true);

      const loadedLogs = storageManager.loadLogs();
      expect(loadedLogs).toHaveLength(2);
      expect(loadedLogs[0].message).toBe('Test message 1');
      expect(loadedLogs[1].message).toBe('Test error message');
    });

    it('should handle empty storage', () => {
      const logs = storageManager.loadLogs();
      expect(logs).toEqual([]);
    });

    it('should clear logs', () => {
      storageManager.saveLogs(mockLogs);
      expect(storageManager.loadLogs()).toHaveLength(2);

      storageManager.clearLogs();
      expect(storageManager.loadLogs()).toEqual([]);
    });

    it('should convert timestamp strings back to Date objects', () => {
      storageManager.saveLogs(mockLogs);
      const loadedLogs = storageManager.loadLogs();
      
      expect(loadedLogs[0].timestamp).toBeInstanceOf(Date);
      expect(loadedLogs[0].timestamp.getTime()).toBe(new Date('2024-01-01T10:00:00Z').getTime());
    });
  });

  describe('Storage Limits', () => {
    it('should apply entry count limits', () => {
      const manyLogs: LogEntry[] = Array.from({ length: 10 }, (_, i) => ({
        id: `${i}`,
        timestamp: new Date(),
        level: 'INFO',
        message: `Message ${i}`,
        serviceName: 'test-service',
        agentId: 'test-agent',
        platform: 'react',
      }));

      storageManager.saveLogs(manyLogs);
      const loadedLogs = storageManager.loadLogs();
      
      // Should keep only the most recent 5 logs (maxEntries = 5)
      expect(loadedLogs).toHaveLength(5);
      expect(loadedLogs[0].message).toBe('Message 5'); // Most recent logs kept
      expect(loadedLogs[4].message).toBe('Message 9');
    });

    it('should apply size limits', () => {
      // Create logs that exceed size limit
      const largeLogs: LogEntry[] = Array.from({ length: 3 }, (_, i) => ({
        id: `${i}`,
        timestamp: new Date(),
        level: 'INFO',
        message: 'A'.repeat(500), // Large message to exceed 1KB limit
        serviceName: 'test-service',
        agentId: 'test-agent',
        platform: 'react',
      }));

      storageManager.saveLogs(largeLogs);
      const loadedLogs = storageManager.loadLogs();
      
      // Should have fewer logs due to size limit
      expect(loadedLogs.length).toBeLessThan(3);
    });

    it('should append to existing logs and apply limits', () => {
      // Save initial logs
      storageManager.saveLogs(mockLogs.slice(0, 1));
      expect(storageManager.loadLogs()).toHaveLength(1);

      // Add more logs
      const moreLogs: LogEntry[] = Array.from({ length: 6 }, (_, i) => ({
        id: `new-${i}`,
        timestamp: new Date(),
        level: 'INFO',
        message: `New message ${i}`,
        serviceName: 'test-service',
        agentId: 'test-agent',
        platform: 'react',
      }));

      storageManager.saveLogs(moreLogs);
      const loadedLogs = storageManager.loadLogs();
      
      // Should have max 5 logs total (1 original + 6 new, limited to 5)
      expect(loadedLogs).toHaveLength(5);
    });
  });

  describe('Storage Statistics', () => {
    it('should provide accurate storage statistics', () => {
      storageManager.saveLogs(mockLogs);
      const stats = storageManager.getStorageStats();

      expect(stats.entryCount).toBe(2);
      expect(stats.sizeBytes).toBeGreaterThan(0);
      expect(stats.maxEntries).toBe(5);
      expect(stats.maxSizeBytes).toBe(1024);
      expect(stats.usagePercent).toBeGreaterThan(0);
      expect(stats.usagePercent).toBeLessThanOrEqual(100);
    });

    it('should show zero stats for empty storage', () => {
      const stats = storageManager.getStorageStats();

      expect(stats.entryCount).toBe(0);
      expect(stats.sizeBytes).toBe(0);
      expect(stats.usagePercent).toBe(0);
    });
  });

  describe('Storage Availability', () => {
    it('should detect localStorage availability', () => {
      expect(storageManager.isStorageAvailable()).toBe(true);
    });

    it('should handle localStorage errors gracefully', () => {
      // Mock localStorage to throw errors
      const originalSetItem = localStorage.setItem;
      localStorage.setItem = vi.fn().mockImplementation(() => {
        throw new Error('Storage quota exceeded');
      });

      const success = storageManager.saveLogs(mockLogs);
      expect(success).toBe(false);

      // Restore original method
      localStorage.setItem = originalSetItem;
    });
  });

  describe('Size Estimation', () => {
    it('should estimate log size accurately', () => {
      const size = storageManager.estimateSize(mockLogs);
      expect(size).toBeGreaterThan(0);
      
      // Size should be reasonable for the test data
      expect(size).toBeGreaterThan(100);
      expect(size).toBeLessThan(10000);
    });

    it('should handle size estimation errors', () => {
      // Mock Blob to throw error
      const originalBlob = global.Blob;
      global.Blob = vi.fn().mockImplementation(() => {
        throw new Error('Blob not supported');
      });

      const size = storageManager.estimateSize(mockLogs);
      expect(size).toBeGreaterThan(0); // Should fallback to string length estimation

      // Restore original Blob
      global.Blob = originalBlob;
    });
  });

  describe('Cleanup Operations', () => {
    it('should cleanup old logs based on age', () => {
      const oldLogs: LogEntry[] = [
        {
          ...mockLogs[0],
          timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2 hours ago
        },
        {
          ...mockLogs[1],
          timestamp: new Date(Date.now() - 30 * 60 * 1000), // 30 minutes ago
        },
      ];

      storageManager.saveLogs(oldLogs);
      
      // Cleanup logs older than 1 hour
      const removedCount = storageManager.cleanupOldLogs(60 * 60 * 1000);
      
      expect(removedCount).toBe(1); // Only the 2-hour-old log should be removed
      
      const remainingLogs = storageManager.loadLogs();
      expect(remainingLogs).toHaveLength(1);
    });

    it('should clear all logs when all are old', () => {
      const oldLogs: LogEntry[] = mockLogs.map(log => ({
        ...log,
        timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2 hours ago
      }));

      storageManager.saveLogs(oldLogs);
      
      // Cleanup logs older than 1 hour
      const removedCount = storageManager.cleanupOldLogs(60 * 60 * 1000);
      
      expect(removedCount).toBe(2);
      expect(storageManager.loadLogs()).toEqual([]);
    });
  });

  describe('Import/Export', () => {
    it('should export logs with metadata', () => {
      storageManager.saveLogs(mockLogs);
      const exported = storageManager.exportLogs();

      expect(exported.logs).toHaveLength(2);
      expect(exported.metadata).toBeDefined();
      expect(exported.timestamp).toBeDefined();
      expect(new Date(exported.timestamp)).toBeInstanceOf(Date);
    });

    it('should import logs successfully', () => {
      const importData = {
        logs: mockLogs,
      };

      const success = storageManager.importLogs(importData);
      expect(success).toBe(true);

      const loadedLogs = storageManager.loadLogs();
      expect(loadedLogs).toHaveLength(2);
      expect(loadedLogs[0].message).toBe('Test message 1');
    });

    it('should handle import errors gracefully', () => {
      // Mock saveLogs to fail
      const originalSaveLogs = storageManager.saveLogs;
      storageManager.saveLogs = vi.fn().mockReturnValue(false);

      const importData = { logs: mockLogs };
      const success = storageManager.importLogs(importData);
      
      expect(success).toBe(false);

      // Restore original method
      storageManager.saveLogs = originalSaveLogs;
    });
  });

  describe('Corrupted Data Handling', () => {
    it('should handle corrupted localStorage data', () => {
      // Manually set corrupted data
      localStorage.setItem('test-logger-buffer', 'invalid json');

      const logs = storageManager.loadLogs();
      expect(logs).toEqual([]);

      // Should have cleared the corrupted data
      expect(localStorage.getItem('test-logger-buffer')).toBeNull();
    });
  });
});