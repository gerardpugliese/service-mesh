package main

import (
	"sync"
	"time"
)

type CircuitBreaker struct {
	failureCount 		int
	lastFailTime 		time.Time
	state 				string // "open" "closed" "half-open"
	failureThreshold 	int // e.g., 3 failures = open
	timeout 			time.Duration // e.g., 5 seconds before retry
	mu 					sync.Mutex // Thread safety
}

// GetState is a helper function to return the 
// current state of a CircuitBreaker
func (cb *CircuitBreaker) GetState() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// GetFailureCount is a helper function to return the 
// current number of failures of a CircuitBreaker
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failureCount
}

// isOpen tells the proxy whether or not an upstream
// server is in an 'open' state or not. 
// It also checks if enough time has passed to transition
// an upstream server into a 'half-open' state
// Returns true if open, false if not.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// If circuit is open, check if enough time has passed to try recovery
	if cb.state == "open" { 
		if time.Since(cb.lastFailTime) >= cb.timeout {
			// Transition to half-open to test recovery
			cb.state = "half-open"
		} 
		return cb.state == "open" // Returns true if still open, false if transitioned
	}
	return false 
}

// RecordFailure is called when a request to an upstream 
// server fails. The failure is recorded in the upstream's
// CircuitBreaker and the state is updated if necessary. 
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Increment failure counter
	cb.failureCount++

	// Set last fail time to  now
	cb.lastFailTime = time.Now()

	// Check if we've gone over failure threshold
	if cb.failureCount == cb.failureThreshold {
		cb.state = "open"
	} 
}

// RecordSuccess is called when a request to an upstream 
// server succeeds. The sucess is recorded in the upstream's
// CircuitBreaker and the state is updated if necessary.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// If we've got a success while in 'half-open'
	// we change state to 'closed'
	if cb.state == "half-open" {
		cb.state = "closed"
	} 

	// Set failure counter back to 0
	cb.failureCount = 0
}
