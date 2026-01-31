package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func main() {
	const iterations = 100000
	line := []byte("Hello, this is a test line!\n")

	// ========== Unbuffered Mode ==========
	f1, _ := os.Create("unbuffered.txt")
	start1 := time.Now()

	for range iterations {
		f1.Write(line)
	}

	f1.Close()
	elapsed1 := time.Since(start1)

	// ========== Buffered Mode ==========
	f2, _ := os.Create("buffered.txt")
	w := bufio.NewWriter(f2)
	start2 := time.Now()

	for range iterations {
		w.WriteString(string(line))
	}

	w.Flush()
	f2.Close()
	elapsed2 := time.Since(start2)

	// ========== Results ==========
	fmt.Println("Unbuffered time:", elapsed1)
	fmt.Println("Buffered time:  ", elapsed2)
	fmt.Printf("Buffered is %.1fx faster\n", float64(elapsed1)/float64(elapsed2))
}