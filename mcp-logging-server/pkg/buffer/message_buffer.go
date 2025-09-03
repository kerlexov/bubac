package buffer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kerlexov/mcp-logging-server/pkg/models"
	"github.com/kerlexov/mcp-logging-server/pkg/storage"
)

// MessageBuffer represents an in-memory buffer for log entries
type MessageBuffer struct {
	storage         storage.LogStorage
	buffer          []models.LogEntry
	mutex           sync.RWMutex
	size            int
	maxBatchSize    int
	flushTimeout    time.Duration
	stopCh          chan struct{}
	flushCh         chan struct{}
	wg              sync.WaitGroup
	recoveryManager RecoveryManager
	metrics         MetricsReporter
}

// RecoveryManager interface for saving pending logs
type RecoveryManager interface {
	SavePendingLogs(logs []models.LogEntry) error
}

// MetricsReporter interface for reporting buffer metrics
type MetricsReporter interface {
	IncrementBufferFlushes()
	IncrementBufferFlushErrors()
	IncrementBufferOverflows()
}

// Config contains configuration for the message buffer
type Config struct {
	Size         int           // Maximum buffer size
	MaxBatchSize int           // Maximum batch size for storage writes
	FlushTimeout time.Duration // Timeout for automatic flushing
}

// Options contains optional dependencies for the message buffer
type Options struct {
	RecoveryManager RecoveryManager
	MetricsReporter MetricsReporter
}

// NewMessageBuffer creates a new message buffer
func NewMessageBuffer(storage storage.LogStorage, config Config) *MessageBuffer {
	return NewMessageBufferWithOptions(storage, config, Options{})
}

// NewMessageBufferWithOptions creates a new message buffer with optional dependencies
func NewMessageBufferWithOptions(storage storage.LogStorage, config Config, options Options) *MessageBuffer {
	return &MessageBuffer{
		storage:         storage,
		buffer:          make([]models.LogEntry, 0, config.Size),
		size:            config.Size,
		maxBatchSize:    config.MaxBatchSize,
		flushTimeout:    config.FlushTimeout,
		stopCh:          make(chan struct{}),
		flushCh:         make(chan struct{}, 1),
		recoveryManager: options.RecoveryManager,
		metrics:         options.MetricsReporter,
	}
}

// Start starts the buffer's background flush routine
func (mb *MessageBuffer) Start(ctx context.Context) {
	mb.wg.Add(1)
	go mb.flushRoutine(ctx)
}

// Stop stops the buffer and flushes any remaining entries
func (mb *MessageBuffer) Stop() error {
	close(mb.stopCh)
	mb.wg.Wait()

	// Save pending logs for recovery if recovery manager is available
	mb.mutex.RLock()
	pendingLogs := make([]models.LogEntry, len(mb.buffer))
	copy(pendingLogs, mb.buffer)
	mb.mutex.RUnlock()

	if mb.recoveryManager != nil && len(pendingLogs) > 0 {
		if err := mb.recoveryManager.SavePendingLogs(pendingLogs); err != nil {
			// Log error but continue with flush
			fmt.Printf("Failed to save pending logs for recovery: %v\n", err)
		}
	}

	// Flush any remaining entries
	return mb.flush(context.Background())
}

// Add adds log entries to the buffer
func (mb *MessageBuffer) Add(entries []models.LogEntry) error {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	for _, entry := range entries {
		// Check if buffer is full
		if len(mb.buffer) >= mb.size {
			// Implement rotation strategy - remove oldest entries
			removeCount := len(mb.buffer) - mb.size + 1
			mb.buffer = mb.buffer[removeCount:]

			// Report buffer overflow
			if mb.metrics != nil {
				mb.metrics.IncrementBufferOverflows()
			}
		}

		mb.buffer = append(mb.buffer, entry)
	}

	// Trigger flush if buffer is getting full or batch size is reached
	if len(mb.buffer) >= mb.maxBatchSize {
		select {
		case mb.flushCh <- struct{}{}:
		default:
			// Channel is full, flush is already scheduled
		}
	}

	return nil
}

// Flush manually flushes the buffer
func (mb *MessageBuffer) Flush() error {
	return mb.flush(context.Background())
}

// GetStats returns buffer statistics
func (mb *MessageBuffer) GetStats() BufferStats {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()

	return BufferStats{
		Size:     len(mb.buffer),
		Capacity: mb.size,
		MaxBatch: mb.maxBatchSize,
	}
}

// BufferStats contains buffer statistics
type BufferStats struct {
	Size     int `json:"size"`
	Capacity int `json:"capacity"`
	MaxBatch int `json:"max_batch"`
}

// flushRoutine runs the background flush routine
func (mb *MessageBuffer) flushRoutine(ctx context.Context) {
	defer mb.wg.Done()

	ticker := time.NewTicker(mb.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mb.stopCh:
			return
		case <-ticker.C:
			// Periodic flush
			if err := mb.flush(ctx); err != nil {
				if mb.metrics != nil {
					mb.metrics.IncrementBufferFlushErrors()
				}
			} else {
				if mb.metrics != nil {
					mb.metrics.IncrementBufferFlushes()
				}
			}
		case <-mb.flushCh:
			// Manual flush trigger
			if err := mb.flush(ctx); err != nil {
				if mb.metrics != nil {
					mb.metrics.IncrementBufferFlushErrors()
				}
			} else {
				if mb.metrics != nil {
					mb.metrics.IncrementBufferFlushes()
				}
			}
		}
	}
}

// flush flushes the buffer to storage
func (mb *MessageBuffer) flush(ctx context.Context) error {
	mb.mutex.Lock()

	if len(mb.buffer) == 0 {
		mb.mutex.Unlock()
		return nil
	}

	// Create batches to avoid overwhelming storage
	var batches [][]models.LogEntry
	for i := 0; i < len(mb.buffer); i += mb.maxBatchSize {
		end := i + mb.maxBatchSize
		if end > len(mb.buffer) {
			end = len(mb.buffer)
		}

		batch := make([]models.LogEntry, end-i)
		copy(batch, mb.buffer[i:end])
		batches = append(batches, batch)
	}

	// Clear buffer after copying
	mb.buffer = mb.buffer[:0]
	mb.mutex.Unlock()

	// Store batches
	for _, batch := range batches {
		if err := mb.storage.Store(ctx, batch); err != nil {
			// On error, try to add entries back to buffer
			mb.mutex.Lock()
			// Only add back if there's space to avoid infinite loops
			if len(mb.buffer)+len(batch) <= mb.size {
				mb.buffer = append(mb.buffer, batch...)
			}
			mb.mutex.Unlock()
			return err
		}
	}

	return nil
}
