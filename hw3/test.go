package main

import (
	"fmt"
	"sync"
)

// KV is a key-value store with version control
// This design is commonly used to implement "Optimistic Locking"
type KV struct {
	mu      sync.Mutex // Mutex ensures only one goroutine can modify data at a time
	value   string     // The stored value
	version int        // Version number, increments after each successful update
}

// put attempts to update the value, but only succeeds if current version equals expectedVersion
// This is an implementation of "Compare-And-Swap" (CAS) operation
// Returns true if update succeeded, false if version mismatch (update rejected)
func (kv *KV) put(newValue string, expectedVersion int) bool {
	kv.mu.Lock()         // Acquire lock, other goroutines must wait
	defer kv.mu.Unlock() // Automatically release lock when function returns

	// Version check: if current version != expected, someone else modified it first
	if kv.version != expectedVersion {
		return false // Reject update to avoid overwriting someone else's changes
	}

	kv.value = newValue // Update the value
	kv.version++        // Increment version
	return true
}

func main() {
	// Initialize: value="init", version=0
	kv := &KV{value: "init", version: 0}

	// WaitGroup is used to wait for all goroutines to complete
	var wg sync.WaitGroup
	wg.Add(2) // Tell WaitGroup we're waiting for 2 goroutines

	// Goroutine 1: Alice tries to update
	go func() {
		defer wg.Done()                          // Notify WaitGroup when done
		ok_1 := kv.put("Alice", 0)               // Expects version=0, wants to set value to "Alice"
		fmt.Println("Alice put result 1:", ok_1)
	}()

	// Goroutine 2: Bob tries to update
	go func() {
		defer wg.Done()
		ok_1 := kv.put("Bob", 0)                 // Also expects version=0, wants to set value to "Bob"
		fmt.Println("Bob put result 1:", ok_1)
	}()

	// Wait for both goroutines to complete
	wg.Wait()
}

/*
================================================================================
QUESTION 1: What is gonna be printed on the terminal? Why?
================================================================================

The result is **UNPREDICTABLE**. There are two possible outcomes:

Case A - Alice acquires the lock first:
  Output: "Alice put result 1: true"
          "Bob put result 1: false"
  Why: Alice checks version=0 ✓, update succeeds, version becomes 1
       Bob checks version=1 != 0 ✗, update fails

Case B - Bob acquires the lock first:
  Output: "Bob put result 1: true"
          "Alice put result 1: false"
  Why: Bob checks version=0 ✓, update succeeds, version becomes 1
       Alice checks version=1 != 0 ✗, update fails

Why is it unpredictable?
- Both goroutines start concurrently, execution order is decided by Go runtime scheduler
- This is a Race Condition
- This type of bug is called a Heisenbug: results may vary between runs, hard to debug

Note: The mutex guarantees data consistency (no simultaneous modifications)
      It does NOT guarantee execution order (who gets the lock first is random)
================================================================================
*/