package main

import (
	"fmt"
	"sync"
)

func main() {
	// Plain map - NOT thread-safe!
	m := make(map[int]int)

	var wg sync.WaitGroup

	// Spawn 50 goroutines
	for g := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each goroutine writes 1000 entries
			for i := range 1000 {
				m[g*1000+i] = i
			}
		}()
	}

	wg.Wait()
	fmt.Println("map length:", len(m))
}