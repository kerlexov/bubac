import { LogEntry } from './types';

export interface StorageConfig {
  maxEntries: number;
  maxSizeBytes: number;
  keyPrefix: string;
}

export class LocalStorageManager {
  private config: StorageConfig;
  private readonly BUFFER_KEY: string;
  private readonly METADATA_KEY: string;

  constructor(config: Partial<StorageConfig> = {}) {
    this.config = {
      maxEntries: 1000,
      maxSizeBytes: 5 * 1024 * 1024, // 5MB
      keyPrefix: 'mcp-logger',
      ...config,
    };

    this.BUFFER_KEY = `${this.config.keyPrefix}-buffer`;
    this.METADATA_KEY = `${this.config.keyPrefix}-metadata`;
  }

  /**
   * Save logs to localStorage with size and count limits
   */
  saveLogs(logs: LogEntry[]): boolean {
    if (!this.isStorageAvailable()) {
      return false;
    }

    try {
      const existingLogs = this.loadLogs();
      const allLogs = [...existingLogs, ...logs];
      
      // Apply limits
      const limitedLogs = this.applyLimits(allLogs);
      
      // Save to localStorage
      const serialized = JSON.stringify(limitedLogs);
      localStorage.setItem(this.BUFFER_KEY, serialized);
      
      // Update metadata
      this.updateMetadata(limitedLogs.length, serialized.length);
      
      return true;
    } catch (error) {
      console.warn('Failed to save logs to localStorage:', error);
      return false;
    }
  }

  /**
   * Load logs from localStorage
   */
  loadLogs(): LogEntry[] {
    if (!this.isStorageAvailable()) {
      return [];
    }

    try {
      const stored = localStorage.getItem(this.BUFFER_KEY);
      if (!stored) {
        return [];
      }

      const logs: LogEntry[] = JSON.parse(stored);
      
      // Validate and convert timestamps
      return logs.map(log => ({
        ...log,
        timestamp: new Date(log.timestamp),
      }));
    } catch (error) {
      console.warn('Failed to load logs from localStorage:', error);
      this.clearLogs(); // Clear corrupted data
      return [];
    }
  }

  /**
   * Clear all stored logs
   */
  clearLogs(): void {
    if (!this.isStorageAvailable()) {
      return;
    }

    try {
      localStorage.removeItem(this.BUFFER_KEY);
      localStorage.removeItem(this.METADATA_KEY);
    } catch (error) {
      console.warn('Failed to clear logs from localStorage:', error);
    }
  }

  /**
   * Get storage statistics
   */
  getStorageStats(): {
    entryCount: number;
    sizeBytes: number;
    maxEntries: number;
    maxSizeBytes: number;
    usagePercent: number;
  } {
    const metadata = this.getMetadata();
    
    return {
      entryCount: metadata.entryCount,
      sizeBytes: metadata.sizeBytes,
      maxEntries: this.config.maxEntries,
      maxSizeBytes: this.config.maxSizeBytes,
      usagePercent: Math.round((metadata.sizeBytes / this.config.maxSizeBytes) * 100),
    };
  }

  /**
   * Check if localStorage is available and has space
   */
  isStorageAvailable(): boolean {
    try {
      const testKey = `${this.config.keyPrefix}-test`;
      localStorage.setItem(testKey, 'test');
      localStorage.removeItem(testKey);
      return true;
    } catch (error) {
      return false;
    }
  }

  /**
   * Estimate the size of logs in bytes
   */
  estimateSize(logs: LogEntry[]): number {
    try {
      return new Blob([JSON.stringify(logs)]).size;
    } catch (error) {
      // Fallback estimation
      return JSON.stringify(logs).length * 2; // Rough estimate for UTF-16
    }
  }

  /**
   * Apply size and count limits to logs
   */
  private applyLimits(logs: LogEntry[]): LogEntry[] {
    let limitedLogs = logs;

    // Apply count limit (keep most recent)
    if (limitedLogs.length > this.config.maxEntries) {
      limitedLogs = limitedLogs.slice(-this.config.maxEntries);
    }

    // Apply size limit (remove oldest until under limit)
    while (limitedLogs.length > 0) {
      const size = this.estimateSize(limitedLogs);
      if (size <= this.config.maxSizeBytes) {
        break;
      }
      
      // Remove oldest 10% of logs
      const removeCount = Math.max(1, Math.floor(limitedLogs.length * 0.1));
      limitedLogs = limitedLogs.slice(removeCount);
    }

    return limitedLogs;
  }

  /**
   * Update storage metadata
   */
  private updateMetadata(entryCount: number, sizeBytes: number): void {
    try {
      const metadata = {
        entryCount,
        sizeBytes,
        lastUpdated: new Date().toISOString(),
      };
      
      localStorage.setItem(this.METADATA_KEY, JSON.stringify(metadata));
    } catch (error) {
      // Ignore metadata update errors
    }
  }

  /**
   * Get storage metadata
   */
  private getMetadata(): { entryCount: number; sizeBytes: number; lastUpdated?: string } {
    try {
      const stored = localStorage.getItem(this.METADATA_KEY);
      if (stored) {
        return JSON.parse(stored);
      }
    } catch (error) {
      // Ignore metadata read errors
    }

    return { entryCount: 0, sizeBytes: 0 };
  }

  /**
   * Cleanup old logs based on age
   */
  cleanupOldLogs(maxAgeMs: number): number {
    const logs = this.loadLogs();
    const cutoffTime = new Date(Date.now() - maxAgeMs);
    
    const filteredLogs = logs.filter(log => 
      new Date(log.timestamp) > cutoffTime
    );

    if (filteredLogs.length < logs.length) {
      const removedCount = logs.length - filteredLogs.length;
      
      if (filteredLogs.length === 0) {
        this.clearLogs();
      } else {
        const serialized = JSON.stringify(filteredLogs);
        localStorage.setItem(this.BUFFER_KEY, serialized);
        this.updateMetadata(filteredLogs.length, serialized.length);
      }
      
      return removedCount;
    }

    return 0;
  }

  /**
   * Export logs for debugging or manual recovery
   */
  exportLogs(): { logs: LogEntry[]; metadata: any; timestamp: string } {
    return {
      logs: this.loadLogs(),
      metadata: this.getMetadata(),
      timestamp: new Date().toISOString(),
    };
  }

  /**
   * Import logs (for testing or recovery)
   */
  importLogs(data: { logs: LogEntry[] }): boolean {
    try {
      return this.saveLogs(data.logs);
    } catch (error) {
      console.warn('Failed to import logs:', error);
      return false;
    }
  }
}