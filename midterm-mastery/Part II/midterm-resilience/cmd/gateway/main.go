package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var httpClient = &http.Client{
	Timeout: 40 * time.Second, // slightly longer than Order Service timeout
}

const orderURL = "http://localhost:8081"

// proxyPost forwards a POST request to a downstream service and returns the response
func proxyPost(c *gin.Context, targetURL string) {
	start := time.Now()

	resp, err := httpClient.Post(targetURL, "application/json", c.Request.Body)
	elapsed := time.Since(start)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   fmt.Sprintf("downstream unreachable: %v", err),
			"latency": elapsed.String(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(resp.StatusCode, gin.H{"raw": string(body), "latency": elapsed.String()})
		return
	}

	c.JSON(resp.StatusCode, gin.H{
		"data":            result,
		"gateway_latency": elapsed.String(),
	})
}

func main() {
	r := gin.Default()

	// Route: Create Order
	r.POST("/orders", func(c *gin.Context) {
		proxyPost(c, orderURL+"/orders")
	})

	// Aggregate Health Check
	r.GET("/health", func(c *gin.Context) {
		services := map[string]string{
			"order":     "http://localhost:8081/health",
			"payment":   "http://localhost:8082/health",
			"inventory": "http://localhost:8083/health",
		}

		healthClient := &http.Client{Timeout: 2 * time.Second}
		results := make(map[string]string)
		allHealthy := true

		for name, url := range services {
			resp, err := healthClient.Get(url)
			if err != nil || resp.StatusCode != http.StatusOK {
				results[name] = "unhealthy"
				allHealthy = false
			} else {
				results[name] = "healthy"
				resp.Body.Close()
			}
		}

		status := http.StatusOK
		overall := "healthy"
		if !allHealthy {
			status = http.StatusServiceUnavailable
			overall = "degraded"
		}

		c.JSON(status, gin.H{
			"status":   overall,
			"services": results,
		})
	})

	fmt.Println("[GATEWAY] Starting on :8080")
	r.Run(":8080")
}