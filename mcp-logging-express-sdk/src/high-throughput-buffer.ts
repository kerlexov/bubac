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

export class HighThroughputBuffer extends EventEmitter {
  private buffers: LogEntry[][] = [];
  private currentBuffer: LogEntry[] = [];
  private maxBufferSize: number;
  private maxBuffers: number;
  private stats: BufferStats;
  private flushInProgress = false;

  constructor(maxBufferSize: number = 1000, maxBuffers: number = 10) {
    super();
    this.maxBufferSize = maxBufferSize;
    this.maxBuffers = maxBuffers;
    this.stats = {
      totalItems: 0,
      bufferSize: 0,
      maxBufferSize,
      flushCount: 0,
      overflowCount: 0,
      lastFlushTime: null
    };
  }

  add(item: LogEntry): boolean {
    this.currentBuffer.push(item);
    this.stats.totalItems++;
    this.stats.bufferSize++;

    // If current buffer is full, rotate it
    if (this.currentBuffer.length >= this.maxBufferSize) {
      this.rotateBuffer();
    }

    return true;
  }

  private rotateBuffer(): void {
    if (this.currentBuffer.length === 0) {
      return;
    }

    // Add current buffer to the buffer pool
    this.buffers.push(this.currentBuffer);
    this.currentBuffer = [];

    // If we have too many buffers, remove the oldest one (overflow)
    if (this.buffers.length > this.maxBuffers) {
      const overflowBuffer = this.buffers.shift();
      if (overflowBuffer) {
        this.stats.overflowCount += overflowBuffer.length;
        this.stats.bufferSize -= overflowBuffer.length;
        this.emit('overflow', overflowBuffer.length);
      }
    }

    this.emit('bufferRotated', this.buffers.length);
  }

  flush(): LogEntry[] {
    if (this.flushInProgress) {
      return [];
    }

    this.flushInProgress = true;

    try {
      // Rotate current buffer if it has items
      if (this.currentBuffer.length > 0) {
        this.rotateBuffer();
      }

      // Get all buffered items
      const allItems: LogEntry[] = [];
      while (this.buffers.length > 0) {
        const buffer = this.buffers.shift();
        if (buffer) {
          allItems.push(...buffer);
        }
      }

      this.stats.bufferSize = 0;
      this.stats.flushCount++;
      this.stats.lastFlushTime = new Date();

      this.emit('flushed', allItems.length);
      return allItems;
    } finally {
      this.flushInProgress = false;
    }
  }

  getStats(): BufferStats {
    return { ...this.stats };
  }

  size(): number {
    return this.stats.bufferSize;
  }

  clear(): void {
    this.buffers = [];
    this.currentBuffer = [];
    this.stats.bufferSize = 0;
    this.emit('cleared');
  }

  // Get items without removing them (for inspection)
  peek(count: number = 10): LogEntry[] {
    const items: LogEntry[] = [];
    
    // Add items from completed buffers first
    for (const buffer of this.buffers) {
      items.push(...buffer);
      if (items.length >= count) {
        break;
      }
    }

    // Add items from current buffer if needed
    if (items.length < count) {
      items.push(...this.currentBuffer.slice(0, count - items.length));
    }

    return items.slice(0, count);
  }
}