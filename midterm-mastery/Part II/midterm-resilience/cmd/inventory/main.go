package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SlowMode struct {
	mu    sync.RWMutex
	slow  bool
	delay time.Duration
}

var mode = &SlowMode{}

func (m *SlowMode) SetSlow(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.slow = true
	m.delay = delay
}

func (m *SlowMode) SetNormal() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.slow = false
}

func (m *SlowMode) Get() (bool, time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.slow, m.delay
}

func main() {
	r := gin.Default()

	// Business Endpoint
	r.POST("/check", func(c *gin.Context) {
		slow, delay := mode.Get()
		if slow {
			fmt.Printf("[INVENTORY] Slow mode — delaying %v\n", delay)
			time.Sleep(delay)
		} else {
			time.Sleep(30 * time.Millisecond) // normal processing time
		}

		available := rand.Float64() > 0.1 // 90% in stock
		fmt.Printf("[INVENTORY] Check complete — available: %v\n", available)
		c.JSON(http.StatusOK, gin.H{
			"available": available,
			"item_id":   fmt.Sprintf("ITEM-%d", rand.Intn(10000)),
		})
	})

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Admin Endpoints
	r.POST("/admin/slow", func(c *gin.Context) {
		mode.SetSlow(5 * time.Second)
		c.JSON(http.StatusOK, gin.H{"message": "slow mode enabled — 5s delay"})
	})

	r.POST("/admin/recover", func(c *gin.Context) {
		mode.SetNormal()
		c.JSON(http.StatusOK, gin.H{"message": "recovered — normal mode"})
	})

	fmt.Println("[INVENTORY] Starting on :8083")
	r.Run(":8083")
}