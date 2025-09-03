package logger

import (
	"context"
	"sync"
	"time"
)

type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitBreakerState
	failureCount     int
	successCount     int
	maxFailures      int
	timeout          time.Duration
	halfOpenMaxCalls int
	lastFailureTime  time.Time
	lastSuccessTime  time.Time
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		maxFailures:      maxFailures,
		timeout:          timeout,
		halfOpenMaxCalls: 5,
	}
}

func (cb *CircuitBreaker) Do(ctx context.Context, fn func() error) error {
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	if state == StateOpen {
		cb.mu.RLock()
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
		} else {
			cb.mu.RUnlock()
			return ErrCircuitBreakerOpen()
		}
	}

	if state == StateHalfOpen {
		cb.mu.RLock()
		if cb.successCount >= cb.halfOpenMaxCalls {
			cb.mu.RUnlock()
			return ErrCircuitBreakerOpen()
		}
		cb.mu.RUnlock()
	}

	err := fn()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		return
	}

	if cb.failureCount >= cb.maxFailures {
		cb.state = StateOpen
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastSuccessTime = time.Now()

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxCalls {
			cb.state = StateClosed
			cb.failureCount = 0
		}
		return
	}

	if cb.state == StateClosed {
		cb.failureCount = 0
	}
}

func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func ErrCircuitBreakerOpen() *Error {
	return &Error{
		Type:    "CIRCUIT_BREAKER_OPEN",
		Message: "circuit breaker is open",
	}
}
