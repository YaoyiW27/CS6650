package failfast

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Checker performs quick health checks before calling a downstream service.
// If the service is known to be down, requests fail immediately instead of waiting.
type Checker struct {
	mu            sync.RWMutex
	serviceURL    string
	healthy       bool
	checkInterval time.Duration
	checkTimeout  time.Duration
	stopCh        chan struct{}
}

// New creates a FailFast checker that periodically pings the service health endpoint.
func New(serviceURL string, checkInterval, checkTimeout time.Duration) *Checker {
	c := &Checker{
		serviceURL:    serviceURL,
		healthy:       true, // assume healthy on start
		checkInterval: checkInterval,
		checkTimeout:  checkTimeout,
		stopCh:        make(chan struct{}),
	}
	go c.backgroundCheck()
	return c
}

// IsHealthy returns whether the downstream service is currently reachable.
func (c *Checker) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthy
}

// Check returns an error immediately if the service is known to be down.
func (c *Checker) Check() error {
	if !c.IsHealthy() {
		return fmt.Errorf("fail-fast: service %s is unreachable", c.serviceURL)
	}
	return nil
}

// backgroundCheck periodically pings the health endpoint.
func (c *Checker) backgroundCheck() {
	client := &http.Client{Timeout: c.checkTimeout}
	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resp, err := client.Get(c.serviceURL + "/health")
			c.mu.Lock()
			if err != nil || resp.StatusCode != http.StatusOK {
				if c.healthy {
					fmt.Printf("[FAILFAST] %s marked UNHEALTHY\n", c.serviceURL)
				}
				c.healthy = false
			} else {
				if !c.healthy {
					fmt.Printf("[FAILFAST] %s marked HEALTHY\n", c.serviceURL)
				}
				c.healthy = true
				resp.Body.Close()
			}
			c.mu.Unlock()

		case <-c.stopCh:
			return
		}
	}
}

// Stop shuts down the background health checker.
func (c *Checker) Stop() {
	close(c.stopCh)
}