package buffer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/your-org/mcp-logging-server/pkg/models"
)

// MockStorage implements storage.LogStorage for testing
type MockStorage struct {
	mutex       sync.Mutex
	storedLogs  []models.LogEntry
	storeError  error
	storeDelay  time.Duration
	storeCalled int
}

func (m *MockStorage) Store(ctx context.Context, logs []models.LogEntry) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.storeCalled++
	
	if m.storeDelay > 0 {
		time.Sleep(m.storeDelay)
	}
	
	if m.storeError != nil {
		return m.storeError
	}
	
	m.storedLogs = append(m.storedLogs, logs...)
	return nil
}

func (m *MockStorage) Query(ctx context.Context, filter models.LogFilter) (*models.LogResult, error) {
	return nil, nil
}

func (m *MockStorage) GetByIDs(ctx context.Context, ids []string) ([]models.LogEntry, error) {
	return nil, nil
}

func (m *MockStorage) GetServices(ctx context.Context) ([]models.ServiceInfo, error) {
	return nil, nil
}

func (m *MockStorage) HealthCheck(ctx context.Context) models.HealthStatus {
	return models.HealthStatus{Status: "healthy"}
}

func (m *MockStorage) Close() error {
	return nil
}

func (m *MockStorage) GetStoredLogs() []models.LogEntry {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	result := make([]models.LogEntry, len(m.storedLogs))
	copy(result, m.storedLogs)
	return result
}

func (m *MockStorage) GetStoreCalled() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.storeCalled
}

func createTestLogEntry(id string) models.LogEntry {
	return models.LogEntry{
		ID:          id,
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}
}

func TestMessageBuffer_Add(t *testing.T) {
	mockStorage := &MockStorage{}
	config := Config{
		Size:         10,
		MaxBatchSize: 5,
		FlushTimeout: 100 * time.Millisecond,
	}
	
	buffer := NewMessageBuffer(mockStorage, config)
	
	// Add single entry
	entry := createTestLogEntry("550e8400-e29b-41d4-a716-446655440000")
	err := buffer.Add([]models.LogEntry{entry})
	if err != nil {
		t.Fatalf("Failed to add entry: %v", err)
	}
	
	stats := buffer.GetStats()
	if stats.Size != 1 {
		t.Errorf("Expected buffer size 1, got %d", stats.Size)
	}
}

func TestMessageBuffer_BufferOverflow(t *testing.T) {
	mockStorage := &MockStorage{}
	config := Config{
		Size:         3,
		MaxBatchSize: 5,
		FlushTimeout: 100 * time.Millisecond,
	}
	
	buffer := NewMessageBuffer(mockStorage, config)
	
	// Add more entries than buffer size
	entries := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440002"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440003"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440004"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440005"),
	}
	
	err := buffer.Add(entries)
	if err != nil {
		t.Fatalf("Failed to add entries: %v", err)
	}
	
	stats := buffer.GetStats()
	if stats.Size != 3 {
		t.Errorf("Expected buffer size 3 (overflow protection), got %d", stats.Size)
	}
}

func TestMessageBuffer_AutoFlush(t *testing.T) {
	mockStorage := &MockStorage{}
	config := Config{
		Size:         10,
		MaxBatchSize: 3,
		FlushTimeout: 50 * time.Millisecond,
	}
	
	buffer := NewMessageBuffer(mockStorage, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	buffer.Start(ctx)
	defer buffer.Stop()
	
	// Add entries that should trigger auto-flush
	entries := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440002"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440003"),
	}
	
	err := buffer.Add(entries)
	if err != nil {
		t.Fatalf("Failed to add entries: %v", err)
	}
	
	// Wait for auto-flush
	time.Sleep(100 * time.Millisecond)
	
	storedLogs := mockStorage.GetStoredLogs()
	if len(storedLogs) != 3 {
		t.Errorf("Expected 3 stored logs, got %d", len(storedLogs))
	}
	
	stats := buffer.GetStats()
	if stats.Size != 0 {
		t.Errorf("Expected buffer to be empty after flush, got size %d", stats.Size)
	}
}

func TestMessageBuffer_PeriodicFlush(t *testing.T) {
	mockStorage := &MockStorage{}
	config := Config{
		Size:         10,
		MaxBatchSize: 5,
		FlushTimeout: 50 * time.Millisecond,
	}
	
	buffer := NewMessageBuffer(mockStorage, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	buffer.Start(ctx)
	defer buffer.Stop()
	
	// Add entries that won't trigger immediate flush
	entries := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440002"),
	}
	
	err := buffer.Add(entries)
	if err != nil {
		t.Fatalf("Failed to add entries: %v", err)
	}
	
	// Wait for periodic flush
	time.Sleep(100 * time.Millisecond)
	
	storedLogs := mockStorage.GetStoredLogs()
	if len(storedLogs) != 2 {
		t.Errorf("Expected 2 stored logs, got %d", len(storedLogs))
	}
}

func TestMessageBuffer_ManualFlush(t *testing.T) {
	mockStorage := &MockStorage{}
	config := Config{
		Size:         10,
		MaxBatchSize: 5,
		FlushTimeout: 1 * time.Second, // Long timeout to avoid auto-flush
	}
	
	buffer := NewMessageBuffer(mockStorage, config)
	
	// Add entries
	entries := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440002"),
	}
	
	err := buffer.Add(entries)
	if err != nil {
		t.Fatalf("Failed to add entries: %v", err)
	}
	
	// Manual flush
	err = buffer.Flush()
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}
	
	storedLogs := mockStorage.GetStoredLogs()
	if len(storedLogs) != 2 {
		t.Errorf("Expected 2 stored logs, got %d", len(storedLogs))
	}
	
	stats := buffer.GetStats()
	if stats.Size != 0 {
		t.Errorf("Expected buffer to be empty after flush, got size %d", stats.Size)
	}
}

func TestMessageBuffer_BatchProcessing(t *testing.T) {
	mockStorage := &MockStorage{}
	config := Config{
		Size:         20,
		MaxBatchSize: 3,
		FlushTimeout: 1 * time.Second,
	}
	
	buffer := NewMessageBuffer(mockStorage, config)
	
	// Add 7 entries (should create 3 batches: 3, 3, 1)
	entries := make([]models.LogEntry, 7)
	for i := 0; i < 7; i++ {
		entries[i] = createTestLogEntry("550e8400-e29b-41d4-a716-44665544000" + string(rune('0'+i)))
	}
	
	err := buffer.Add(entries)
	if err != nil {
		t.Fatalf("Failed to add entries: %v", err)
	}
	
	err = buffer.Flush()
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}
	
	storedLogs := mockStorage.GetStoredLogs()
	if len(storedLogs) != 7 {
		t.Errorf("Expected 7 stored logs, got %d", len(storedLogs))
	}
	
	// Check that Store was called multiple times (batching)
	storeCalled := mockStorage.GetStoreCalled()
	if storeCalled < 2 {
		t.Errorf("Expected Store to be called at least 2 times for batching, got %d", storeCalled)
	}
}

func TestMessageBuffer_ErrorHandling(t *testing.T) {
	mockStorage := &MockStorage{
		storeError: errors.New("storage error"),
	}
	config := Config{
		Size:         10,
		MaxBatchSize: 5,
		FlushTimeout: 1 * time.Second,
	}
	
	buffer := NewMessageBuffer(mockStorage, config)
	
	// Add entries
	entries := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440002"),
	}
	
	err := buffer.Add(entries)
	if err != nil {
		t.Fatalf("Failed to add entries: %v", err)
	}
	
	// Flush should return error
	err = buffer.Flush()
	if err == nil {
		t.Error("Expected flush to return error")
	}
	
	// Entries should be added back to buffer on error
	stats := buffer.GetStats()
	if stats.Size != 2 {
		t.Errorf("Expected buffer to contain 2 entries after failed flush, got %d", stats.Size)
	}
}

func TestMessageBuffer_ConcurrentAccess(t *testing.T) {
	mockStorage := &MockStorage{}
	config := Config{
		Size:         100,
		MaxBatchSize: 10,
		FlushTimeout: 100 * time.Millisecond,
	}
	
	buffer := NewMessageBuffer(mockStorage, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	buffer.Start(ctx)
	defer buffer.Stop()
	
	// Concurrent adds
	var wg sync.WaitGroup
	numGoroutines := 10
	entriesPerGoroutine := 5
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < entriesPerGoroutine; j++ {
				entry := createTestLogEntry("550e8400-e29b-41d4-a716-44665544" + string(rune('0'+goroutineID)) + string(rune('0'+j)))
				err := buffer.Add([]models.LogEntry{entry})
				if err != nil {
					t.Errorf("Failed to add entry: %v", err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Wait for flushes to complete
	time.Sleep(200 * time.Millisecond)
	
	// Final flush to ensure all entries are stored
	buffer.Flush()
	
	storedLogs := mockStorage.GetStoredLogs()
	expectedTotal := numGoroutines * entriesPerGoroutine
	if len(storedLogs) != expectedTotal {
		t.Errorf("Expected %d stored logs, got %d", expectedTotal, len(storedLogs))
	}
}