package logger

import (
	"testing"
	"time"
)

func TestMemoryBuffer(t *testing.T) {
	buffer := newMemoryBuffer(3)

	if buffer.Size() != 0 {
		t.Errorf("Expected buffer size 0, got %d", buffer.Size())
	}

	if buffer.IsFull() {
		t.Error("Expected buffer not to be full")
	}

	entry1 := LogEntry{
		ID:        "1",
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   "Test message 1",
	}

	entry2 := LogEntry{
		ID:        "2",
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   "Test message 2",
	}

	entry3 := LogEntry{
		ID:        "3",
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   "Test message 3",
	}

	err := buffer.Add(entry1)
	if err != nil {
		t.Errorf("Expected no error adding entry, got %v", err)
	}

	if buffer.Size() != 1 {
		t.Errorf("Expected buffer size 1, got %d", buffer.Size())
	}

	buffer.Add(entry2)
	buffer.Add(entry3)

	if !buffer.IsFull() {
		t.Error("Expected buffer to be full")
	}

	if buffer.Size() != 3 {
		t.Errorf("Expected buffer size 3, got %d", buffer.Size())
	}

	entries, err := buffer.Flush()
	if err != nil {
		t.Errorf("Expected no error flushing, got %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	if buffer.Size() != 0 {
		t.Errorf("Expected buffer size 0 after flush, got %d", buffer.Size())
	}

	entry4 := LogEntry{
		ID:        "4",
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   "Test message 4",
	}

	buffer.Add(entry1)
	buffer.Add(entry2)
	buffer.Add(entry3)
	buffer.Add(entry4)

	if buffer.Size() != 3 {
		t.Errorf("Expected buffer size 3 after overflow, got %d", buffer.Size())
	}

	entries, _ = buffer.Flush()
	if entries[0].ID != "2" {
		t.Errorf("Expected first entry ID to be '2' after rotation, got %s", entries[0].ID)
	}
}
