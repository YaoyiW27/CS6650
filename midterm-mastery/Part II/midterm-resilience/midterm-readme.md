# Resilient Microservices Demo: Crash & Recovery

A distributed microservices system demonstrating common failure scenarios and three resilience patterns — **Fail Fast**, **Circuit Breaker**, and **Bulkhead** — implemented in Go, showcasing how each pattern addresses cascading failures in a microservices architecture.

## Architecture

```
                    ┌─────────────────┐
   Locust ────────▶ │   API Gateway   │ :8080
  (load test)       │    (Go/Gin)     │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Order Service  │ :8081
                    │    (Go/Gin)     │
                    └───┬─────────┬───┘
                        │         │
               ┌────────▼───┐ ┌───▼────────┐
               │  Payment   │ │ Inventory  │
               │  Service   │ │  Service   │
               │   :8082    │ │   :8083    │
               │  (crashes) │ │  (slows)   │
               └────────────┘ └────────────┘
```

### Services

- **API Gateway** (`:8080`) — Entry point. Routes incoming requests to Order Service.
- **Order Service** (`:8081`) — Orchestrates order processing. Calls Payment and Inventory services. Implements all resilience patterns.
- **Payment Service** (`:8082`) — Simulates payment processing. Supports fault injection (crash, hang, flaky).
- **Inventory Service** (`:8083`) — Simulates inventory checks. Supports slow mode.

## Fault Injection

Payment Service exposes admin endpoints to simulate failures on demand:

| Endpoint | Effect |
|---|---|
| `POST /admin/crash` | Service stops responding (process exit) |
| `POST /admin/hang` | Responses delayed by 30s |
| `POST /admin/flaky?rate=0.5` | 50% of requests fail randomly |
| `POST /admin/recover` | Restore normal operation |

## Failure Scenarios & Resilience Patterns

### Phase 1: No Resilience (Baseline)

**Scenario:** Payment Service hangs (30s delay). No protection in place.

**What happens:** Order Service uses a 35s HTTP timeout. Every request blocks waiting for Payment to respond. With 100 concurrent users, all threads are occupied waiting — the entire system grinds to a halt.

### Phase 2: Fail Fast

**Pattern:** A background health checker pings each downstream service every 2 seconds. If a service is unreachable, incoming requests are rejected immediately (0ms) instead of waiting for a timeout.

**What happens:** Health checker detects Payment is down within 2-4 seconds. All subsequent requests return instantly with a clear error. System resources are freed immediately.

### Phase 3: Circuit Breaker

**Pattern:** Tracks the failure rate of downstream calls. After 3 consecutive failures, the circuit "opens" — all requests are short-circuited with a fallback response for a 10-second cooldown. After cooldown, one probe request is allowed ("half-open") to test recovery.

**States:** Closed (normal) → Open (blocking) → Half-Open (testing) → Closed (recovered)

**What happens:** Under flaky conditions (50% failure rate), the circuit breaker absorbs failures and returns graceful fallback responses (HTTP 202 "pending"). The system maintains high throughput and Inventory calls are unaffected.

### Phase 4: Bulkhead + Circuit Breaker + Fail Fast (Combined)

**Pattern:** All three patterns layered together. Additionally, Payment and Inventory calls are isolated into separate goroutine pools (max 10 concurrent each) and executed in parallel.

**What happens:** Fail Fast catches known-down services instantly. Circuit Breaker handles intermittent failures. Bulkhead ensures Payment failures cannot exhaust resources needed by Inventory. Parallel execution reduces latency when both services are healthy.

## Load Test Results

Load testing performed with **Locust**: 100 concurrent users, 10 users/sec spawn rate, ~60 seconds per phase.

### Results Summary

| Phase | Median (ms) | p95 (ms) | p99 (ms) | RPS | Failure % | Total Requests |
|---|---|---|---|---|---|---|
| **1 - No Resilience** | 30,033 | 30,000 | 30,000 | 7.3 | 0% | 500 |
| **2 - Fail Fast** | 1 | 4 | 20 | 330.3 | 100% | 25,979 |
| **3 - Circuit Breaker** | 32 | 52 | 66 | 295.7 | 0% | 26,705 |
| **4 - Bulkhead + All** | 1 | 4 | 18 | 332.1 | 100% | 21,390 |

### Key Observations

- **Phase 1 → Phase 2:** Latency dropped from 30,033ms to 1ms. Throughput increased 45x (7.3 → 330.3 RPS). Fail Fast prevents resource exhaustion by rejecting requests to unhealthy services immediately.

- **Phase 3 (Circuit Breaker):** Under 50% flaky conditions, achieved 0% Locust failures with 295.7 RPS. The circuit breaker returns HTTP 202 (pending) as a graceful fallback, so the system degrades gracefully rather than failing hard.

- **Phase 4 (All Combined):** Best overall performance — 1ms median, 332.1 RPS. Parallel execution of Inventory and Payment calls, combined with all three resilience patterns, provides maximum protection and efficiency.

- **Failure % context:** Phase 2 and 4 show 100% Locust failures because Payment is completely down (hang mode) — all requests correctly return error responses. The key insight is that these errors return in 1ms instead of 30 seconds, keeping the system responsive.

## Quick Start

```bash
# Initialize project
go mod init midterm-resilience
go get github.com/gin-gonic/gin

# Run all 4 services (in separate terminals)
go run cmd/gateway/main.go
go run cmd/order/main.go
go run cmd/payment/main.go
go run cmd/inventory/main.go

# Test normal order
curl -X POST http://localhost:8080/orders

# Inject fault and test each phase
curl -X POST http://localhost:8082/admin/hang
curl -X POST http://localhost:8080/orders/phase1   # 30s hang
curl -X POST http://localhost:8080/orders/phase2   # instant fail
curl -X POST http://localhost:8080/orders/phase3   # circuit breaker
curl -X POST http://localhost:8080/orders/phase4   # bulkhead + all

# View resilience stats
curl http://localhost:8080/stats

# Recover
curl -X POST http://localhost:8082/admin/recover

# Load test a specific phase
pip install locust
locust -f locustfile.py --host=http://localhost:8080 --tags phase1
# Open http://localhost:8089, set 100 users / 10 spawn rate
```