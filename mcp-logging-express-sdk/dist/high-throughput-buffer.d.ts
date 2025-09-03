/// <reference types="node" />
import { LogEntry } from './types';
import { EventEmitter } from 'events';
export interface BufferStats {
    totalItems: number;
    bufferSize: number;
    maxBufferSize: number;
    flushCount: number;
    overflowCount: number;
    lastFlushTime: Date | null;
}
export declare class HighThroughputBuffer extends EventEmitter {
    private buffers;
    private currentBuffer;
    private maxBufferSize;
    private maxBuffers;
    private stats;
    private flushInProgress;
    constructor(maxBufferSize?: number, maxBuffers?: number);
    add(item: LogEntry): boolean;
    private rotateBuffer;
    flush(): LogEntry[];
    getStats(): BufferStats;
    size(): number;
    clear(): void;
    peek(count?: number): LogEntry[];
}
//# sourceMappingURL=high-throughput-buffer.d.ts.map