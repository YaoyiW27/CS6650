package main

import (
	"fmt"
	"runtime"
	"time"
)

func pingpong(iterations int) time.Duration {
	ping := make(chan struct{})
	pong := make(chan struct{})
	done := make(chan struct{})

	// Goroutine 1: receives on ping, sends on pong
	go func() {
		for range iterations {
			<-ping
			pong <- struct{}{}
		}
	}()

	// Goroutine 2: sends on ping, receives on pong
	go func() {
		for range iterations {
			ping <- struct{}{}
			<-pong
		}
		done <- struct{}{}
	}()

	start := time.Now()
	<-done
	return time.Since(start)
}

func main() {
	const iterations = 1000000

	// ========== Single Thread Mode ==========
	runtime.GOMAXPROCS(1)
	elapsed1 := pingpong(iterations)
	avgSwitch1 := float64(elapsed1.Nanoseconds()) / float64(iterations*2)

	// ========== Multi Thread Mode ==========
	runtime.GOMAXPROCS(runtime.NumCPU())
	elapsed2 := pingpong(iterations)
	avgSwitch2 := float64(elapsed2.Nanoseconds()) / float64(iterations*2)

	// ========== Results ==========
	fmt.Println("=== Single Thread (GOMAXPROCS=1) ===")
	fmt.Printf("Total time: %v\n", elapsed1)
	fmt.Printf("Avg switch time: %.0f ns\n", avgSwitch1)

	fmt.Println("\n=== Multi Thread (GOMAXPROCS=NumCPU) ===")
	fmt.Printf("Total time: %v\n", elapsed2)
	fmt.Printf("Avg switch time: %.0f ns\n", avgSwitch2)
}