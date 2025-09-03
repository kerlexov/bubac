import { HighThroughputBuffer } from '../high-throughput-buffer';
import { LogEntry, LogLevel } from '../types';

describe('HighThroughputBuffer', () => {
  let buffer: HighThroughputBuffer;

  beforeEach(() => {
    buffer = new HighThroughputBuffer(3, 2); // Small sizes for testing
  });

  const createLogEntry = (message: string): LogEntry => ({
    id: `test-${Date.now()}-${Math.random()}`,
    timestamp: new Date(),
    level: LogLevel.INFO,
    message,
    serviceName: 'test',
    agentId: 'test',
    platform: 'test'
  });

  test('should add items to buffer', () => {
    const entry = createLogEntry('test message');
    buffer.add(entry);
    
    expect(buffer.size()).toBe(1);
  });

  test('should rotate buffer when full', () => {
    let rotationCount = 0;
    buffer.on('bufferRotated', () => rotationCount++);

    // Fill the buffer to trigger rotation
    buffer.add(createLogEntry('message 1'));
    buffer.add(createLogEntry('message 2'));
    buffer.add(createLogEntry('message 3')); // Should trigger rotation

    expect(rotationCount).toBe(1);
  });

  test('should handle overflow when too many buffers', () => {
    let overflowCount = 0;
    buffer.on('overflow', (count) => overflowCount += count);

    // Fill multiple buffers to trigger overflow
    for (let i = 0; i < 12; i++) { // 4 full buffers (3 items each)
      buffer.add(createLogEntry(`message ${i}`));
    }

    expect(overflowCount).toBeGreaterThan(0);
  });

  test('should flush all items', () => {
    buffer.add(createLogEntry('message 1'));
    buffer.add(createLogEntry('message 2'));
    buffer.add(createLogEntry('message 3'));
    buffer.add(createLogEntry('message 4')); // Triggers rotation

    const flushed = buffer.flush();
    
    expect(flushed.length).toBe(4);
    expect(buffer.size()).toBe(0);
  });

  test('should provide stats', () => {
    buffer.add(createLogEntry('test'));
    const stats = buffer.getStats();

    expect(stats.totalItems).toBe(1);
    expect(stats.bufferSize).toBe(1);
    expect(stats.maxBufferSize).toBe(3);
  });

  test('should peek items without removing them', () => {
    buffer.add(createLogEntry('message 1'));
    buffer.add(createLogEntry('message 2'));

    const peeked = buffer.peek(1);
    
    expect(peeked.length).toBe(1);
    expect(buffer.size()).toBe(2); // Items should still be in buffer
  });

  test('should clear all items', () => {
    buffer.add(createLogEntry('test'));
    buffer.clear();

    expect(buffer.size()).toBe(0);
    expect(buffer.flush()).toEqual([]);
  });
});