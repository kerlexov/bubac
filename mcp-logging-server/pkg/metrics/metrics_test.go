package metrics

import (
	"testing"
	"time"
)

func TestMetrics_Counters(t *testing.T) {
	metrics := NewMetrics()
	
	// Test initial state
	snapshot := metrics.GetSnapshot()
	if snapshot.RequestsTotal != 0 {
		t.Errorf("Expected initial RequestsTotal to be 0, got %d", snapshot.RequestsTotal)
	}
	
	// Test incrementing counters
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsSuccessful()
	metrics.IncrementLogsIngested(5)
	metrics.IncrementLogsBuffered(3)
	
	snapshot = metrics.GetSnapshot()
	if snapshot.RequestsTotal != 1 {
		t.Errorf("Expected RequestsTotal to be 1, got %d", snapshot.RequestsTotal)
	}
	if snapshot.RequestsSuccessful != 1 {
		t.Errorf("Expected RequestsSuccessful to be 1, got %d", snapshot.RequestsSuccessful)
	}
	if snapshot.LogsIngested != 5 {
		t.Errorf("Expected LogsIngested to be 5, got %d", snapshot.LogsIngested)
	}
	if snapshot.LogsBuffered != 3 {
		t.Errorf("Expected LogsBuffered to be 3, got %d", snapshot.LogsBuffered)
	}
}

func TestMetrics_SuccessRate(t *testing.T) {
	metrics := NewMetrics()
	
	// Test with no requests
	snapshot := metrics.GetSnapshot()
	if snapshot.SuccessRate != 0.0 {
		t.Errorf("Expected SuccessRate to be 0.0 with no requests, got %f", snapshot.SuccessRate)
	}
	
	// Test with all successful requests
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsSuccessful()
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsSuccessful()
	
	snapshot = metrics.GetSnapshot()
	if snapshot.SuccessRate != 100.0 {
		t.Errorf("Expected SuccessRate to be 100.0, got %f", snapshot.SuccessRate)
	}
	
	// Test with mixed success/failure
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsFailed()
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsFailed()
	
	snapshot = metrics.GetSnapshot()
	expectedRate := 50.0 // 2 successful out of 4 total
	if snapshot.SuccessRate != expectedRate {
		t.Errorf("Expected SuccessRate to be %f, got %f", expectedRate, snapshot.SuccessRate)
	}
}

func TestMetrics_ErrorRate(t *testing.T) {
	metrics := NewMetrics()
	
	// Test with no requests
	snapshot := metrics.GetSnapshot()
	if snapshot.ErrorRate != 0.0 {
		t.Errorf("Expected ErrorRate to be 0.0 with no requests, got %f", snapshot.ErrorRate)
	}
	
	// Test with some failed requests
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsSuccessful()
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsFailed()
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsFailed()
	
	snapshot = metrics.GetSnapshot()
	expectedRate := 66.67 // 2 failed out of 3 total, rounded
	if snapshot.ErrorRate < 66.0 || snapshot.ErrorRate > 67.0 {
		t.Errorf("Expected ErrorRate to be around %f, got %f", expectedRate, snapshot.ErrorRate)
	}
}

func TestMetrics_Uptime(t *testing.T) {
	metrics := NewMetrics()
	
	// Wait a bit to ensure uptime is measurable
	time.Sleep(1100 * time.Millisecond)
	
	snapshot := metrics.GetSnapshot()
	if snapshot.UptimeSeconds <= 0 {
		t.Errorf("Expected UptimeSeconds to be positive, got %d", snapshot.UptimeSeconds)
	}
	
	if snapshot.ServerStartTime.IsZero() {
		t.Error("Expected ServerStartTime to be set")
	}
}

func TestMetrics_LastRequestTime(t *testing.T) {
	metrics := NewMetrics()
	
	// Initially, last request time should be zero
	snapshot := metrics.GetSnapshot()
	if !snapshot.LastRequestTime.IsZero() {
		t.Error("Expected LastRequestTime to be zero initially")
	}
	
	// After incrementing requests, last request time should be set
	metrics.IncrementRequestsTotal()
	
	snapshot = metrics.GetSnapshot()
	if snapshot.LastRequestTime.IsZero() {
		t.Error("Expected LastRequestTime to be set after incrementing requests")
	}
}

func TestMetrics_BufferMetrics(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.IncrementBufferFlushes()
	metrics.IncrementBufferFlushErrors()
	metrics.IncrementBufferOverflows()
	
	snapshot := metrics.GetSnapshot()
	if snapshot.BufferFlushes != 1 {
		t.Errorf("Expected BufferFlushes to be 1, got %d", snapshot.BufferFlushes)
	}
	if snapshot.BufferFlushErrors != 1 {
		t.Errorf("Expected BufferFlushErrors to be 1, got %d", snapshot.BufferFlushErrors)
	}
	if snapshot.BufferOverflows != 1 {
		t.Errorf("Expected BufferOverflows to be 1, got %d", snapshot.BufferOverflows)
	}
}

func TestMetrics_ValidationAndStorageErrors(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.IncrementValidationErrors()
	metrics.IncrementStorageErrors()
	
	snapshot := metrics.GetSnapshot()
	if snapshot.ValidationErrors != 1 {
		t.Errorf("Expected ValidationErrors to be 1, got %d", snapshot.ValidationErrors)
	}
	if snapshot.StorageErrors != 1 {
		t.Errorf("Expected StorageErrors to be 1, got %d", snapshot.StorageErrors)
	}
}

func TestMetrics_Reset(t *testing.T) {
	metrics := NewMetrics()
	
	// Set some values
	metrics.IncrementRequestsTotal()
	metrics.IncrementRequestsSuccessful()
	metrics.IncrementLogsIngested(10)
	
	// Verify values are set
	snapshot := metrics.GetSnapshot()
	if snapshot.RequestsTotal == 0 {
		t.Error("Expected RequestsTotal to be non-zero before reset")
	}
	
	// Reset and verify
	metrics.Reset()
	
	snapshot = metrics.GetSnapshot()
	if snapshot.RequestsTotal != 0 {
		t.Errorf("Expected RequestsTotal to be 0 after reset, got %d", snapshot.RequestsTotal)
	}
	if snapshot.RequestsSuccessful != 0 {
		t.Errorf("Expected RequestsSuccessful to be 0 after reset, got %d", snapshot.RequestsSuccessful)
	}
	if snapshot.LogsIngested != 0 {
		t.Errorf("Expected LogsIngested to be 0 after reset, got %d", snapshot.LogsIngested)
	}
	if snapshot.LastRequestTime.IsZero() == false {
		t.Error("Expected LastRequestTime to be zero after reset")
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewMetrics()
	
	// Test concurrent access
	done := make(chan bool, 2)
	
	go func() {
		for i := 0; i < 100; i++ {
			metrics.IncrementRequestsTotal()
			metrics.IncrementRequestsSuccessful()
		}
		done <- true
	}()
	
	go func() {
		for i := 0; i < 100; i++ {
			metrics.IncrementLogsIngested(1)
			metrics.GetSnapshot()
		}
		done <- true
	}()
	
	// Wait for both goroutines to complete
	<-done
	<-done
	
	snapshot := metrics.GetSnapshot()
	if snapshot.RequestsTotal != 100 {
		t.Errorf("Expected RequestsTotal to be 100, got %d", snapshot.RequestsTotal)
	}
	if snapshot.RequestsSuccessful != 100 {
		t.Errorf("Expected RequestsSuccessful to be 100, got %d", snapshot.RequestsSuccessful)
	}
	if snapshot.LogsIngested != 100 {
		t.Errorf("Expected LogsIngested to be 100, got %d", snapshot.LogsIngested)
	}
}