// The primary mechanism for managing state in Go is
// communication over channels. We saw this for example
// with [worker pools](worker-pools). There are a few other
// options for managing state though. Here we'll
// look at using the `sync/atomic` package for _atomic
// counters_ accessed by multiple goroutines.

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func main() {

	// We'll use an atomic integer type to represent our
	// (always-positive) counter.
	var ops atomic.Uint64

	// A regular integer for comparison - this will have race conditions!
	var regularOps uint64

	// Expected value: 50 goroutines * 1000 increments = 50000
	expected := uint64(50 * 1000)

	// A WaitGroup will help us wait for all goroutines
	// to finish their work.
	var wg sync.WaitGroup

	// We'll start 50 goroutines that each increment the
	// counter exactly 1000 times.
	for range 50 {
		wg.Go(func() {
			for range 1000 {
				// To atomically increment the counter we use `Add`.
				ops.Add(1)

				// This is NOT atomic - will cause race condition!
				regularOps++
			}
		})
	}

	// Wait until all the goroutines are done.
	wg.Wait()

	// Here no goroutines are writing to 'ops', but using
	// `Load` it's safe to atomically read a value even while
	// other goroutines are (atomically) updating it.
	fmt.Println("expected:", expected)
	fmt.Println("atomic ops:", ops.Load())
	fmt.Println("regular ops:", regularOps)
}