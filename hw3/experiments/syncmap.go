package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var m sync.Map

	var wg sync.WaitGroup

	start := time.Now()

	// Spawn 50 goroutines
	for g := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 1000 {
				m.Store(g*1000+i, i)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Count entries using Range
	count := 0
	m.Range(func(key, value any) bool {
		count++
		return true
	})

	fmt.Println("map length:", count)
	fmt.Println("time taken:", elapsed)
}