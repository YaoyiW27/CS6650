package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var httpClient = &http.Client{
	Timeout: 35 * time.Second, // long timeout — no resilience yet
}

const (
	paymentURL   = "http://localhost:8082"
	inventoryURL = "http://localhost:8083"
)

type OrderRequest struct {
	ItemID   string  `json:"item_id"`
	Quantity int     `json:"quantity"`
	Amount   float64 `json:"amount"`
}

type OrderResponse struct {
	OrderID          string `json:"order_id"`
	Status           string `json:"status"`
	PaymentResult    any    `json:"payment_result,omitempty"`
	InventoryResult  any    `json:"inventory_result,omitempty"`
	Error            string `json:"error,omitempty"`
	PaymentLatency   string `json:"payment_latency"`
	InventoryLatency string `json:"inventory_latency"`
	TotalLatency     string `json:"total_latency"`
}

// callService makes a POST request to a downstream service and returns the parsed response
func callService(url string) (map[string]any, time.Duration, error) {
	start := time.Now()

	resp, err := httpClient.Post(url, "application/json", nil)
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

func main() {
	r := gin.Default()

	// Create Order — No Resilience (Phase 1)
	r.POST("/orders", func(c *gin.Context) {
		totalStart := time.Now()
		orderID := fmt.Sprintf("ORD-%d", rand.Intn(100000))

		resp := OrderResponse{OrderID: orderID}

		// Step 1: Call Inventory Service (sequential — waits for completion)
		invResult, invLatency, invErr := callService(inventoryURL + "/check")
		resp.InventoryLatency = invLatency.String()

		if invErr != nil {
			resp.Status = "failed"
			resp.Error = fmt.Sprintf("inventory check failed: %v", invErr)
			resp.TotalLatency = time.Since(totalStart).String()
			c.JSON(http.StatusInternalServerError, resp)
			return
		}
		resp.InventoryResult = invResult

		// Step 2: Call Payment Service (sequential — waits for completion)
		payResult, payLatency, payErr := callService(paymentURL + "/pay")
		resp.PaymentLatency = payLatency.String()

		if payErr != nil {
			resp.Status = "failed"
			resp.Error = fmt.Sprintf("payment failed: %v", payErr)
			resp.TotalLatency = time.Since(totalStart).String()
			c.JSON(http.StatusInternalServerError, resp)
			return
		}
		resp.PaymentResult = payResult

		// Both succeeded
		resp.Status = "confirmed"
		resp.TotalLatency = time.Since(totalStart).String()
		c.JSON(http.StatusOK, resp)
	})

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "order"})
	})

	fmt.Println("[ORDER] Starting on :8081 (Phase 1 — No Resilience)")
	r.Run(":8081")
}