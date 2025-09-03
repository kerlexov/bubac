package buffer

import (
	"sync"

	"github.com/kerlexov/mcp-logging-go-sdk/pkg/logger"
)

type MemoryBuffer struct {
	entries []logger.LogEntry
	maxSize int
	mu      sync.Mutex
}

func NewMemoryBuffer(maxSize int) *MemoryBuffer {
	return &MemoryBuffer{
		entries: make([]logger.LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

func (b *MemoryBuffer) Add(entry logger.LogEntry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) >= b.maxSize {
		copy(b.entries, b.entries[1:])
		b.entries = b.entries[:len(b.entries)-1]
	}

	b.entries = append(b.entries, entry)
	return nil
}

func (b *MemoryBuffer) Flush() ([]logger.LogEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) == 0 {
		return nil, nil
	}

	entries := make([]logger.LogEntry, len(b.entries))
	copy(entries, b.entries)
	b.entries = b.entries[:0]

	return entries, nil
}

func (b *MemoryBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

func (b *MemoryBuffer) IsFull() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries) >= b.maxSize
}

func (b *MemoryBuffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = nil
	return nil
}
