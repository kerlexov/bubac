import { LogEntry } from './types';
export interface StorageConfig {
    maxEntries: number;
    maxSizeBytes: number;
    keyPrefix: string;
}
export declare class LocalStorageManager {
    private config;
    private readonly BUFFER_KEY;
    private readonly METADATA_KEY;
    constructor(config?: Partial<StorageConfig>);
    /**
     * Save logs to localStorage with size and count limits
     */
    saveLogs(logs: LogEntry[]): boolean;
    /**
     * Load logs from localStorage
     */
    loadLogs(): LogEntry[];
    /**
     * Clear all stored logs
     */
    clearLogs(): void;
    /**
     * Get storage statistics
     */
    getStorageStats(): {
        entryCount: number;
        sizeBytes: number;
        maxEntries: number;
        maxSizeBytes: number;
        usagePercent: number;
    };
    /**
     * Check if localStorage is available and has space
     */
    isStorageAvailable(): boolean;
    /**
     * Estimate the size of logs in bytes
     */
    estimateSize(logs: LogEntry[]): number;
    /**
     * Apply size and count limits to logs
     */
    private applyLimits;
    /**
     * Update storage metadata
     */
    private updateMetadata;
    /**
     * Get storage metadata
     */
    private getMetadata;
    /**
     * Cleanup old logs based on age
     */
    cleanupOldLogs(maxAgeMs: number): number;
    /**
     * Export logs for debugging or manual recovery
     */
    exportLogs(): {
        logs: LogEntry[];
        metadata: any;
        timestamp: string;
    };
    /**
     * Import logs (for testing or recovery)
     */
    importLogs(data: {
        logs: LogEntry[];
    }): boolean;
}
