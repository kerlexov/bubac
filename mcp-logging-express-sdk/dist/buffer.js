"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Buffer = void 0;
class Buffer {
    constructor(maxSize) {
        this.items = [];
        this.maxSize = maxSize;
    }
    add(item) {
        this.items.push(item);
        // If buffer is full, remove oldest items (FIFO)
        if (this.items.length > this.maxSize) {
            this.items = this.items.slice(-this.maxSize);
        }
    }
    flush() {
        const items = [...this.items];
        this.items = [];
        return items;
    }
    size() {
        return this.items.length;
    }
    isFull() {
        return this.items.length >= this.maxSize;
    }
    clear() {
        this.items = [];
    }
}
exports.Buffer = Buffer;
//# sourceMappingURL=buffer.js.map