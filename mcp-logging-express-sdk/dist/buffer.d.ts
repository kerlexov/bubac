export declare class Buffer<T> {
    private items;
    private maxSize;
    constructor(maxSize: number);
    add(item: T): void;
    flush(): T[];
    size(): number;
    isFull(): boolean;
    clear(): void;
}
//# sourceMappingURL=buffer.d.ts.map