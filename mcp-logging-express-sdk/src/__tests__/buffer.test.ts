import { Buffer } from '../buffer';

describe('Buffer', () => {
  let buffer: Buffer<string>;

  beforeEach(() => {
    buffer = new Buffer<string>(3);
  });

  test('should add items to buffer', () => {
    buffer.add('item1');
    buffer.add('item2');
    
    expect(buffer.size()).toBe(2);
  });

  test('should flush all items', () => {
    buffer.add('item1');
    buffer.add('item2');
    
    const items = buffer.flush();
    
    expect(items).toEqual(['item1', 'item2']);
    expect(buffer.size()).toBe(0);
  });

  test('should handle buffer overflow with FIFO', () => {
    buffer.add('item1');
    buffer.add('item2');
    buffer.add('item3');
    buffer.add('item4'); // Should remove item1
    
    const items = buffer.flush();
    
    expect(items).toEqual(['item2', 'item3', 'item4']);
  });

  test('should report when buffer is full', () => {
    expect(buffer.isFull()).toBe(false);
    
    buffer.add('item1');
    buffer.add('item2');
    buffer.add('item3');
    
    expect(buffer.isFull()).toBe(true);
  });

  test('should clear buffer', () => {
    buffer.add('item1');
    buffer.add('item2');
    
    buffer.clear();
    
    expect(buffer.size()).toBe(0);
    expect(buffer.flush()).toEqual([]);
  });

  test('should handle empty buffer flush', () => {
    const items = buffer.flush();
    
    expect(items).toEqual([]);
    expect(buffer.size()).toBe(0);
  });
});