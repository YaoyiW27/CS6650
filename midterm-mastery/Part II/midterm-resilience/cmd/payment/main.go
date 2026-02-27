package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// FaultMode controls how the service behaves
type FaultMode struct {
	mu        sync.RWMutex
	mode      string  // "normal", "crash", "hang", "flaky"
	flakyRate float64 // probability of failure in flaky mode
	hangDelay time.Duration
}

var fault = &FaultMode{mode: "normal"}

func (f *FaultMode) Set(mode string, opts ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mode = mode
	if mode == "flaky" && len(opts) > 0 {
		if rate, ok := opts[0].(float64); ok {
			f.flakyRate = rate
		}
	}
	if mode == "hang" {
		f.hangDelay = 30 * time.Second
		if len(opts) > 0 {
			if d, ok := opts[0].(time.Duration); ok {
				f.hangDelay = d
			}
		}
	}
}

func (f *FaultMode) Get() (string, float64, time.Duration) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.mode, f.flakyRate, f.hangDelay
}

func main() {
	r := gin.Default()

	// Business Endpoint
	r.POST("/pay", func(c *gin.Context) {
		mode, flakyRate, hangDelay := fault.Get()

		switch mode {
		case "crash":
			fmt.Println("[PAYMENT] Simulating crash! Exiting process...")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "service crashed"})
			go func() {
				time.Sleep(100 * time.Millisecond)
				os.Exit(1)
			}()
			return

		case "hang":
			fmt.Printf("[PAYMENT] Simulating hang for %v...\n", hangDelay)
			time.Sleep(hangDelay)
			c.JSON(http.StatusOK, gin.H{"status": "paid", "note": "delayed"})
			return

		case "flaky":
			if rand.Float64() < flakyRate {
				fmt.Println("[PAYMENT] Flaky failure!")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "random failure"})
				return
			}
		}

		// Normal processing
		time.Sleep(50 * time.Millisecond) // simulate work
		fmt.Println("[PAYMENT] Payment processed successfully")
		c.JSON(http.StatusOK, gin.H{
			"status":     "paid",
			"payment_id": fmt.Sprintf("PAY-%d", rand.Intn(100000)),
		})
	})

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		mode, _, _ := fault.Get()
		if mode == "crash" || mode == "hang" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "mode": mode})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "mode": mode})
	})

	// Fault Injection Admin Endpoints
	r.POST("/admin/crash", func(c *gin.Context) {
		fault.Set("crash")
		c.JSON(http.StatusOK, gin.H{"message": "crash mode enabled — next /pay call will kill the process"})
	})

	r.POST("/admin/hang", func(c *gin.Context) {
		delay := 30 * time.Second
		if d := c.Query("delay"); d != "" {
			if sec, err := strconv.Atoi(d); err == nil {
				delay = time.Duration(sec) * time.Second
			}
		}
		fault.Set("hang", delay)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("hang mode enabled — %v delay", delay)})
	})

	r.POST("/admin/flaky", func(c *gin.Context) {
		rate := 0.5
		if r := c.Query("rate"); r != "" {
			if parsed, err := strconv.ParseFloat(r, 64); err == nil {
				rate = parsed
			}
		}
		fault.Set("flaky", rate)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("flaky mode enabled — %.0f%% failure rate", rate*100)})
	})

	r.POST("/admin/recover", func(c *gin.Context) {
		fault.Set("normal")
		c.JSON(http.StatusOK, gin.H{"message": "recovered — normal mode"})
	})

	r.GET("/admin/status", func(c *gin.Context) {
		mode, rate, delay := fault.Get()
		c.JSON(http.StatusOK, gin.H{"mode": mode, "flaky_rate": rate, "hang_delay": delay.String()})
	})

	fmt.Println("[PAYMENT] Starting on :8082")
	r.Run(":8082")
}