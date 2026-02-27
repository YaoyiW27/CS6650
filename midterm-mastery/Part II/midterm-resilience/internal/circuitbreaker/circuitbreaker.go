package circuitbreaker

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type State int

const (
	Closed   State = iota // normal — requests pass through
	Open                  // tripped — requests short-circuited
	HalfOpen              // testing — allow one request to probe recovery
)

func (s State) String() string {
	switch s {
	case Closed:
		return "CLOSED"
	case Open:
		return "OPEN"
	case HalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

var ErrCircuitOpen = errors.New("circuit breaker is OPEN — request blocked")

// CircuitBreaker tracks failures and short-circuits calls when a threshold is exceeded.
type CircuitBreaker struct {
	mu               sync.Mutex
	name             string
	state            State
	failureCount     int
	successCount     int
	failureThreshold int           // failures needed to trip the circuit
	successThreshold int           // successes in half-open to close the circuit
	cooldownPeriod   time.Duration // how long to stay open before half-open
	lastFailureTime  time.Time
}

// New creates a CircuitBreaker with the given parameters.
func New(name string, failureThreshold, successThreshold int, cooldownPeriod time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             name,
		state:            Closed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		cooldownPeriod:   cooldownPeriod,
	}
	fmt.Printf("[CIRCUIT-BREAKER] %s initialized — threshold=%d, cooldown=%v\n",
		name, failureThreshold, cooldownPeriod)
	return cb
}

// Execute runs the given function through the circuit breaker.
// If the circuit is open, it returns ErrCircuitOpen immediately.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	switch cb.state {
	case Open:
		// Check if cooldown has elapsed → transition to half-open
		if time.Since(cb.lastFailureTime) > cb.cooldownPeriod {
			fmt.Printf("[CIRCUIT-BREAKER] %s: OPEN → HALF-OPEN (cooldown elapsed)\n", cb.name)
			cb.state = HalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
			return cb.doExecute(fn)
		}
		cb.mu.Unlock()
		return ErrCircuitOpen

	case HalfOpen:
		cb.mu.Unlock()
		return cb.doExecute(fn)

	default: // Closed
		cb.mu.Unlock()
		return cb.doExecute(fn)
	}
}

func (cb *CircuitBreaker) doExecute(fn func() error) error {
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.state == HalfOpen {
			// Any failure in half-open → back to open
			fmt.Printf("[CIRCUIT-BREAKER] %s: HALF-OPEN → OPEN (probe failed)\n", cb.name)
			cb.state = Open
		} else if cb.failureCount >= cb.failureThreshold {
			fmt.Printf("[CIRCUIT-BREAKER] %s: CLOSED → OPEN (failures=%d)\n", cb.name, cb.failureCount)
			cb.state = Open
		}
		return err
	}

	// Success
	if cb.state == HalfOpen {
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			fmt.Printf("[CIRCUIT-BREAKER] %s: HALF-OPEN → CLOSED (recovered!)\n", cb.name)
			cb.state = Closed
			cb.failureCount = 0
			cb.successCount = 0
		}
	} else {
		// Reset failure count on success in closed state
		cb.failureCount = 0
	}

	return nil
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// GetStats returns current counters for monitoring.
func (cb *CircuitBreaker) GetStats() map[string]any {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return map[string]any{
		"name":           cb.name,
		"state":          cb.state.String(),
		"failure_count":  cb.failureCount,
		"success_count":  cb.successCount,
		"last_failure":   cb.lastFailureTime.Format(time.RFC3339),
	}
}