package logger

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(2, 1*time.Second)

	// Test closed state - should allow calls
	if cb.GetState() != StateClosed {
		t.Errorf("Expected circuit breaker to be closed initially")
	}

	// Simulate failures to open the circuit breaker
	failureFunc := func() error {
		return errors.New("simulated failure")
	}

	// First failure
	err := cb.Do(context.Background(), failureFunc)
	if err == nil {
		t.Error("Expected error from failure function")
	}
	if cb.GetState() != StateClosed {
		t.Error("Expected circuit breaker to remain closed after first failure")
	}

	// Second failure - should open the circuit breaker
	err = cb.Do(context.Background(), failureFunc)
	if err == nil {
		t.Error("Expected error from failure function")
	}
	if cb.GetState() != StateOpen {
		t.Error("Expected circuit breaker to be open after max failures")
	}

	// Third call should fail immediately due to open circuit breaker
	err = cb.Do(context.Background(), func() error { return nil })
	if err == nil {
		t.Error("Expected circuit breaker open error")
	}
}

func TestRetryLogic(t *testing.T) {
	config := RetryConfig{
		InitialInterval:     10 * time.Millisecond,
		MaxInterval:         100 * time.Millisecond,
		MaxElapsedTime:      1 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	}

	retryer := newRetryer(config)
	
	attempts := 0
	testFunc := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("simulated failure")
		}
		return nil
	}

	ctx := context.Background()
	err := retryer.Do(ctx, testFunc)
	
	if err != nil {
		t.Errorf("Expected retry to succeed eventually, got error: %v", err)
	}
	
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestBufferRotation(t *testing.T) {
	buffer := newMemoryBuffer(2)
	
	entry1 := LogEntry{ID: "1", Message: "Message 1", Level: LogLevelInfo, Timestamp: time.Now()}
	entry2 := LogEntry{ID: "2", Message: "Message 2", Level: LogLevelInfo, Timestamp: time.Now()}
	entry3 := LogEntry{ID: "3", Message: "Message 3", Level: LogLevelInfo, Timestamp: time.Now()}
	
	// Add entries up to capacity
	buffer.Add(entry1)
	buffer.Add(entry2)
	
	if buffer.Size() != 2 {
		t.Errorf("Expected buffer size 2, got %d", buffer.Size())
	}
	
	// Add one more entry - should trigger rotation
	buffer.Add(entry3)
	
	if buffer.Size() != 2 {
		t.Errorf("Expected buffer size to remain 2 after rotation, got %d", buffer.Size())
	}
	
	// Flush and check that oldest entry was rotated out
	entries, err := buffer.Flush()
	if err != nil {
		t.Errorf("Expected no error flushing, got %v", err)
	}
	
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries after flush, got %d", len(entries))
	}
	
	// First entry should be entry2 (entry1 was rotated out)
	if entries[0].ID != "2" {
		t.Errorf("Expected first entry ID to be '2', got '%s'", entries[0].ID)
	}
	
	// Second entry should be entry3
	if entries[1].ID != "3" {
		t.Errorf("Expected second entry ID to be '3', got '%s'", entries[1].ID)
	}
}