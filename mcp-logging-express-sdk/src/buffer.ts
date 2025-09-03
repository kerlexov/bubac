export class Buffer<T> {
  private items: T[] = [];
  private maxSize: number;

  constructor(maxSize: number) {
    this.maxSize = maxSize;
  }

  add(item: T): void {
    this.items.push(item);
    
    // If buffer is full, remove oldest items (FIFO)
    if (this.items.length > this.maxSize) {
      this.items = this.items.slice(-this.maxSize);
    }
  }

  flush(): T[] {
    const items = [...this.items];
    this.items = [];
    return items;
  }

  size(): number {
    return this.items.length;
  }

  isFull(): boolean {
    return this.items.length >= this.maxSize;
  }

  clear(): void {
    this.items = [];
  }
}