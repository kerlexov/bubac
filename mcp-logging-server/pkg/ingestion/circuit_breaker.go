package ingestion

import (
	"errors"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for resilience
type CircuitBreaker struct {
	mutex           sync.RWMutex
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	timeout         time.Duration
	maxFailures     int
	resetTimeout    time.Duration
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, timeout time.Duration, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        StateClosed,
		maxFailures:  maxFailures,
		timeout:      timeout,
		resetTimeout: resetTimeout,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	// Check if circuit should transition from open to half-open
	if cb.state == StateOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
		cb.state = StateHalfOpen
		cb.successCount = 0
	}
	
	// Reject requests if circuit is open
	if cb.state == StateOpen {
		return errors.New("circuit breaker is open")
	}
	
	// Execute the function
	err := fn()
	
	if err != nil {
		cb.onFailure()
		return err
	}
	
	cb.onSuccess()
	return nil
}

// onFailure handles failure cases
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	if cb.failureCount >= cb.maxFailures {
		cb.state = StateOpen
	}
}

// onSuccess handles success cases
func (cb *CircuitBreaker) onSuccess() {
	cb.failureCount = 0
	
	if cb.state == StateHalfOpen {
		cb.successCount++
		// After a few successful requests in half-open state, close the circuit
		if cb.successCount >= 3 {
			cb.state = StateClosed
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return CircuitBreakerStats{
		State:           cb.state,
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailureTime: cb.lastFailureTime,
	}
}

// CircuitBreakerStats contains statistics about the circuit breaker
type CircuitBreakerStats struct {
	State           CircuitBreakerState `json:"state"`
	FailureCount    int                 `json:"failure_count"`
	SuccessCount    int                 `json:"success_count"`
	LastFailureTime time.Time           `json:"last_failure_time"`
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastFailureTime = time.Time{}
}