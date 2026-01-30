package main

import (
	"fmt"
	"sync"
	"time"
)

// SafeMap wraps a map with a mutex
type SafeMap struct {
	mu sync.Mutex
	m  map[int]int
}

func main() {
	sm := SafeMap{m: make(map[int]int)}

	var wg sync.WaitGroup

	start := time.Now()

	// Spawn 50 goroutines
	for g := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 1000 {
				// Lock before writing
				sm.mu.Lock()
				sm.m[g*1000+i] = i
				sm.mu.Unlock()
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Println("map length:", len(sm.m))
	fmt.Println("time taken:", elapsed)
}