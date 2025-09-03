package logger

import (
	"sync"
)

type memoryBuffer struct {
	entries []LogEntry
	maxSize int
	mu      sync.Mutex
}

func newMemoryBuffer(maxSize int) *memoryBuffer {
	return &memoryBuffer{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

func (b *memoryBuffer) Add(entry LogEntry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) >= b.maxSize {
		copy(b.entries, b.entries[1:])
		b.entries = b.entries[:len(b.entries)-1]
	}

	b.entries = append(b.entries, entry)
	return nil
}

func (b *memoryBuffer) Flush() ([]LogEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) == 0 {
		return nil, nil
	}

	entries := make([]LogEntry, len(b.entries))
	copy(entries, b.entries)
	b.entries = b.entries[:0]

	return entries, nil
}

func (b *memoryBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

func (b *memoryBuffer) IsFull() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries) >= b.maxSize
}

func (b *memoryBuffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = nil
	return nil
}
