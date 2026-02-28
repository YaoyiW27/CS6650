package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"midterm-resilience/internal/bulkhead"
	"midterm-resilience/internal/circuitbreaker"
	"midterm-resilience/internal/failfast"

	"github.com/gin-gonic/gin"
)

const (
	paymentURL   = "http://localhost:8082"
	inventoryURL = "http://localhost:8083"
)

// --- HTTP clients with different timeouts ---

var noResilienceClient = &http.Client{
	Timeout: 35 * time.Second, // Phase 1: long timeout, no protection
}

var resilientClient = &http.Client{
	Timeout: 3 * time.Second, // Phase 2-4: short timeout for fail-fast
}

// --- Resilience components ---

var (
	paymentFailFast   *failfast.Checker
	inventoryFailFast *failfast.Checker

	paymentCB   *circuitbreaker.CircuitBreaker
	inventoryCB *circuitbreaker.CircuitBreaker

	paymentBH   *bulkhead.Bulkhead
	inventoryBH *bulkhead.Bulkhead
)

func init() {
	// Fail Fast: health check every 2s, 1s timeout per check
	paymentFailFast = failfast.New(paymentURL, 2*time.Second, 1*time.Second)
	inventoryFailFast = failfast.New(inventoryURL, 2*time.Second, 1*time.Second)

	// Circuit Breaker: open after 3 failures, close after 2 successes, 10s cooldown
	paymentCB = circuitbreaker.New("payment", 3, 2, 10*time.Second)
	inventoryCB = circuitbreaker.New("inventory", 3, 2, 10*time.Second)

	// Bulkhead: 10 concurrent workers per service, 500ms wait timeout
	paymentBH = bulkhead.New("payment", 10, 500*time.Millisecond)
	inventoryBH = bulkhead.New("inventory", 10, 500*time.Millisecond)
}

// --- Shared helpers ---

type OrderResponse struct {
	OrderID          string `json:"order_id"`
	Status           string `json:"status"`
	Phase            string `json:"phase"`
	PaymentResult    any    `json:"payment_result,omitempty"`
	InventoryResult  any    `json:"inventory_result,omitempty"`
	Error            string `json:"error,omitempty"`
	PaymentLatency   string `json:"payment_latency"`
	InventoryLatency string `json:"inventory_latency"`
	TotalLatency     string `json:"total_latency"`
}

func callService(client *http.Client, url string) (map[string]any, time.Duration, error) {
	start := time.Now()
	resp, err := client.Post(url, "application/json", nil)
	elapsed := time.Since(start)
	if err != nil {
		return nil, elapsed, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, elapsed, fmt.Errorf("read body failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, elapsed, fmt.Errorf("service returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, elapsed, fmt.Errorf("json parse failed: %w", err)
	}
	return result, elapsed, nil
}

func newOrderID() string {
	return fmt.Sprintf("ORD-%d", rand.Intn(100000))
}

// ============================================================
// Phase 1: No Resilience
// ============================================================
func phase1Handler(c *gin.Context) {
	totalStart := time.Now()
	resp := OrderResponse{OrderID: newOrderID(), Phase: "1-no-resilience"}

	// Sequential calls with long timeout — everything blocks
	invResult, invLat, invErr := callService(noResilienceClient, inventoryURL+"/check")
	resp.InventoryLatency = invLat.String()
	if invErr != nil {
		resp.Status = "failed"
		resp.Error = fmt.Sprintf("inventory: %v", invErr)
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.InventoryResult = invResult

	payResult, payLat, payErr := callService(noResilienceClient, paymentURL+"/pay")
	resp.PaymentLatency = payLat.String()
	if payErr != nil {
		resp.Status = "failed"
		resp.Error = fmt.Sprintf("payment: %v", payErr)
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.PaymentResult = payResult

	resp.Status = "confirmed"
	resp.TotalLatency = time.Since(totalStart).String()
	c.JSON(http.StatusOK, resp)
}

// ============================================================
// Phase 2: Fail Fast
// ============================================================
func phase2Handler(c *gin.Context) {
	totalStart := time.Now()
	resp := OrderResponse{OrderID: newOrderID(), Phase: "2-fail-fast"}

	// Check health before calling — fail immediately if down
	if err := paymentFailFast.Check(); err != nil {
		resp.Status = "failed"
		resp.Error = err.Error()
		resp.PaymentLatency = "0ms (fail-fast)"
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}
	if err := inventoryFailFast.Check(); err != nil {
		resp.Status = "failed"
		resp.Error = err.Error()
		resp.InventoryLatency = "0ms (fail-fast)"
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}

	// Calls with short timeout
	invResult, invLat, invErr := callService(resilientClient, inventoryURL+"/check")
	resp.InventoryLatency = invLat.String()
	if invErr != nil {
		resp.Status = "failed"
		resp.Error = fmt.Sprintf("inventory: %v", invErr)
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.InventoryResult = invResult

	payResult, payLat, payErr := callService(resilientClient, paymentURL+"/pay")
	resp.PaymentLatency = payLat.String()
	if payErr != nil {
		resp.Status = "failed"
		resp.Error = fmt.Sprintf("payment: %v", payErr)
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.PaymentResult = payResult

	resp.Status = "confirmed"
	resp.TotalLatency = time.Since(totalStart).String()
	c.JSON(http.StatusOK, resp)
}

// ============================================================
// Phase 3: Circuit Breaker
// ============================================================
func phase3Handler(c *gin.Context) {
	totalStart := time.Now()
	resp := OrderResponse{OrderID: newOrderID(), Phase: "3-circuit-breaker"}

	// Inventory call through circuit breaker
	var invResult map[string]any
	var invLat time.Duration
	invErr := inventoryCB.Execute(func() error {
		var err error
		invResult, invLat, err = callService(resilientClient, inventoryURL+"/check")
		return err
	})
	resp.InventoryLatency = invLat.String()
	if invErr != nil {
		resp.Status = "failed"
		resp.Error = fmt.Sprintf("inventory: %v", invErr)
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}
	resp.InventoryResult = invResult

	// Payment call through circuit breaker
	var payResult map[string]any
	var payLat time.Duration
	payErr := paymentCB.Execute(func() error {
		var err error
		payResult, payLat, err = callService(resilientClient, paymentURL+"/pay")
		return err
	})
	resp.PaymentLatency = payLat.String()
	if payErr != nil {
		// Circuit breaker provides fallback
		resp.Status = "pending"
		resp.Error = fmt.Sprintf("payment: %v (order queued for retry)", payErr)
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusAccepted, resp) // 202 — accepted but not completed
		return
	}
	resp.PaymentResult = payResult

	resp.Status = "confirmed"
	resp.TotalLatency = time.Since(totalStart).String()
	c.JSON(http.StatusOK, resp)
}

// ============================================================
// Phase 4: Bulkhead + Circuit Breaker + Fail Fast (all combined)
// ============================================================
func phase4Handler(c *gin.Context) {
	totalStart := time.Now()
	resp := OrderResponse{OrderID: newOrderID(), Phase: "4-bulkhead"}

	// Fail fast check first
	if err := paymentFailFast.Check(); err != nil {
		resp.Status = "failed"
		resp.Error = err.Error()
		resp.PaymentLatency = "0ms (fail-fast)"
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}

	// Parallel calls through isolated bulkheads
	var invResult, payResult map[string]any
	var invLat, payLat time.Duration
	var invErr, payErr error
	var wg sync.WaitGroup

	// Inventory — bulkhead + circuit breaker
	wg.Add(1)
	go func() {
		defer wg.Done()
		invErr = inventoryBH.Execute(func() error {
			return inventoryCB.Execute(func() error {
				var err error
				invResult, invLat, err = callService(resilientClient, inventoryURL+"/check")
				return err
			})
		})
	}()

	// Payment — bulkhead + circuit breaker
	wg.Add(1)
	go func() {
		defer wg.Done()
		payErr = paymentBH.Execute(func() error {
			return paymentCB.Execute(func() error {
				var err error
				payResult, payLat, err = callService(resilientClient, paymentURL+"/pay")
				return err
			})
		})
	}()

	wg.Wait()

	resp.InventoryLatency = invLat.String()
	resp.PaymentLatency = payLat.String()

	if invErr != nil {
		resp.Status = "failed"
		resp.Error = fmt.Sprintf("inventory: %v", invErr)
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusServiceUnavailable, resp)
		return
	}
	resp.InventoryResult = invResult

	if payErr != nil {
		resp.Status = "pending"
		resp.Error = fmt.Sprintf("payment: %v (order queued for retry)", payErr)
		resp.InventoryResult = invResult
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusAccepted, resp)
		return
	}
	resp.PaymentResult = payResult

	resp.Status = "confirmed"
	resp.TotalLatency = time.Since(totalStart).String()
	c.JSON(http.StatusOK, resp)
}

// ============================================================
// Main
// ============================================================
func main() {
	r := gin.Default()

	// Phase endpoints
	r.POST("/orders/phase1", phase1Handler)
	r.POST("/orders/phase2", phase2Handler)
	r.POST("/orders/phase3", phase3Handler)
	r.POST("/orders/phase4", phase4Handler)

	// Default route — alias to phase 1
	r.POST("/orders", phase1Handler)

	// Health & stats
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "order"})
	})

	r.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"circuit_breakers": gin.H{
				"payment":   paymentCB.GetStats(),
				"inventory": inventoryCB.GetStats(),
			},
			"bulkheads": gin.H{
				"payment":   paymentBH.GetStats(),
				"inventory": inventoryBH.GetStats(),
			},
			"failfast": gin.H{
				"payment_healthy":   paymentFailFast.IsHealthy(),
				"inventory_healthy": inventoryFailFast.IsHealthy(),
			},
		})
	})

	fmt.Println("[ORDER] Starting on :8081 — All phases available")
	fmt.Println("  POST /orders/phase1  — No resilience")
	fmt.Println("  POST /orders/phase2  — Fail Fast")
	fmt.Println("  POST /orders/phase3  — Circuit Breaker")
	fmt.Println("  POST /orders/phase4  — Bulkhead + CB + FF (all combined)")
	fmt.Println("  GET  /stats          — View resilience component stats")
	r.Run(":8081")
}