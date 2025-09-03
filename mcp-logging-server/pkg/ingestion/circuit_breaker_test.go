package ingestion

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second, 60*time.Second)
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cb.GetState())
	}
	
	stats := cb.GetStats()
	if stats.FailureCount != 0 {
		t.Errorf("Expected initial failure count to be 0, got %d", stats.FailureCount)
	}
}

func TestCircuitBreaker_SuccessfulExecution(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second, 60*time.Second)
	
	err := cb.Execute(func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain Closed after success, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_FailureHandling(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second, 60*time.Second)
	
	// Execute failing function multiple times
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return errors.New("test error")
		})
		
		if err == nil {
			t.Error("Expected error, got nil")
		}
		
		if cb.GetState() != StateClosed {
			t.Errorf("Expected state to remain Closed after %d failures, got %v", i+1, cb.GetState())
		}
	}
	
	// Third failure should open the circuit
	err := cb.Execute(func() error {
		return errors.New("test error")
	})
	
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open after 3 failures, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_OpenStateRejectsRequests(t *testing.T) {
	cb := NewCircuitBreaker(1, 30*time.Second, 60*time.Second)
	
	// Cause one failure to open the circuit
	cb.Execute(func() error {
		return errors.New("test error")
	})
	
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open, got %v", cb.GetState())
	}
	
	// Next request should be rejected
	err := cb.Execute(func() error {
		return nil // This function should not be called
	})
	
	if err == nil {
		t.Error("Expected circuit breaker error, got nil")
	}
	
	if err.Error() != "circuit breaker is open" {
		t.Errorf("Expected circuit breaker error message, got %v", err.Error())
	}
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	cb := NewCircuitBreaker(1, 30*time.Second, 100*time.Millisecond) // Short reset timeout for testing
	
	// Cause failure to open circuit
	cb.Execute(func() error {
		return errors.New("test error")
	})
	
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open, got %v", cb.GetState())
	}
	
	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)
	
	// Next execution should transition to half-open
	err := cb.Execute(func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error in half-open state, got %v", err)
	}
	
	// After successful executions, should transition to closed
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return nil
		})
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after successful executions, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(1, 30*time.Second, 60*time.Second)
	
	// Cause failure to open circuit
	cb.Execute(func() error {
		return errors.New("test error")
	})
	
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be Open, got %v", cb.GetState())
	}
	
	// Reset circuit breaker
	cb.Reset()
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after reset, got %v", cb.GetState())
	}
	
	stats := cb.GetStats()
	if stats.FailureCount != 0 {
		t.Errorf("Expected failure count to be 0 after reset, got %d", stats.FailureCount)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second, 60*time.Second)
	
	// Execute some failures
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return errors.New("test error")
		})
	}
	
	stats := cb.GetStats()
	if stats.State != StateClosed {
		t.Errorf("Expected state to be Closed, got %v", stats.State)
	}
	
	if stats.FailureCount != 2 {
		t.Errorf("Expected failure count to be 2, got %d", stats.FailureCount)
	}
	
	if stats.LastFailureTime.IsZero() {
		t.Error("Expected LastFailureTime to be set")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(10, 30*time.Second, 60*time.Second)
	
	// Test concurrent access
	done := make(chan bool, 2)
	
	go func() {
		for i := 0; i < 50; i++ {
			cb.Execute(func() error {
				return nil
			})
		}
		done <- true
	}()
	
	go func() {
		for i := 0; i < 50; i++ {
			cb.Execute(func() error {
				if i%10 == 0 {
					return errors.New("test error")
				}
				return nil
			})
		}
		done <- true
	}()
	
	// Wait for both goroutines to complete
	<-done
	<-done
	
	// Circuit should still be closed since we didn't exceed the failure threshold
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be Closed after concurrent access, got %v", cb.GetState())
	}
}