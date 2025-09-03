"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.HighThroughputBuffer = void 0;
const events_1 = require("events");
class HighThroughputBuffer extends events_1.EventEmitter {
    constructor(maxBufferSize = 1000, maxBuffers = 10) {
        super();
        this.buffers = [];
        this.currentBuffer = [];
        this.flushInProgress = false;
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
    add(item) {
        this.currentBuffer.push(item);
        this.stats.totalItems++;
        this.stats.bufferSize++;
        // If current buffer is full, rotate it
        if (this.currentBuffer.length >= this.maxBufferSize) {
            this.rotateBuffer();
        }
        return true;
    }
    rotateBuffer() {
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
    flush() {
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
            const allItems = [];
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
        }
        finally {
            this.flushInProgress = false;
        }
    }
    getStats() {
        return { ...this.stats };
    }
    size() {
        return this.stats.bufferSize;
    }
    clear() {
        this.buffers = [];
        this.currentBuffer = [];
        this.stats.bufferSize = 0;
        this.emit('cleared');
    }
    // Get items without removing them (for inspection)
    peek(count = 10) {
        const items = [];
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
exports.HighThroughputBuffer = HighThroughputBuffer;
//# sourceMappingURL=high-throughput-buffer.js.map