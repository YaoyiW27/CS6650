package bulkhead

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrBulkheadFull = errors.New("bulkhead: pool is full — request rejected")

// Bulkhead isolates downstream service calls into bounded goroutine pools.
// When one pool fills up, it doesn't affect others.
type Bulkhead struct {
	name       string
	semaphore  chan struct{}
	maxWorkers int
	timeout    time.Duration

	mu         sync.Mutex
	active     int
	rejected   int
	completed  int
}

// New creates a Bulkhead with the given concurrency limit and wait timeout.
func New(name string, maxWorkers int, timeout time.Duration) *Bulkhead {
	b := &Bulkhead{
		name:       name,
		semaphore:  make(chan struct{}, maxWorkers),
		maxWorkers: maxWorkers,
		timeout:    timeout,
	}
	fmt.Printf("[BULKHEAD] %s initialized — max_workers=%d, timeout=%v\n",
		name, maxWorkers, timeout)
	return b
}

// Execute runs the function within the bulkhead's bounded pool.
// If the pool is full and no slot opens within the timeout, it returns ErrBulkheadFull.
func (b *Bulkhead) Execute(fn func() error) error {
	// Try to acquire a slot
	select {
	case b.semaphore <- struct{}{}:
		// Got a slot
		b.mu.Lock()
		b.active++
		b.mu.Unlock()

		// Execute and release
		err := fn()

		<-b.semaphore
		b.mu.Lock()
		b.active--
		b.completed++
		b.mu.Unlock()

		return err

	case <-time.After(b.timeout):
		// Pool is full, reject immediately
		b.mu.Lock()
		b.rejected++
		b.mu.Unlock()
		fmt.Printf("[BULKHEAD] %s: REJECTED — pool full (%d/%d active)\n",
			b.name, b.active, b.maxWorkers)
		return ErrBulkheadFull
	}
}

// GetStats returns current pool statistics.
func (b *Bulkhead) GetStats() map[string]any {
	b.mu.Lock()
	defer b.mu.Unlock()
	return map[string]any{
		"name":        b.name,
		"max_workers": b.maxWorkers,
		"active":      b.active,
		"rejected":    b.rejected,
		"completed":   b.completed,
	}
}