# Resilient Microservices Demo: Crash & Recovery

A distributed microservices system demonstrating common failure scenarios and resilience patterns (Fail Fast, Circuit Breaker, Bulkhead) in a Go-based e-commerce order processing pipeline, deployed on AWS EC2.

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
- **Order Service** (`:8081`) — Orchestrates order processing. Calls Payment and Inventory services.
- **Payment Service** (`:8082`) — Simulates payment processing. Will be configured to crash/hang to trigger failures.
- **Inventory Service** (`:8083`) — Simulates inventory checks. Will be configured to slow down under load.

## Failure Scenarios & Resilience Patterns

### Phase 1: No Resilience (Baseline)

**Scenario:** Payment Service crashes or hangs. No protection in place.

**Expected behavior:**
- Order Service threads block waiting for Payment responses
- Request queue backs up, all requests time out
- Inventory Service is healthy but becomes unreachable because Order Service is overwhelmed
- Cascading failure — entire system becomes unresponsive

### Phase 2: Fail Fast

**Scenario:** Same failure, but Order Service implements fail-fast checks.

**Pattern:** Before calling a downstream service, quickly verify it's reachable (health check / timeout). If not, return an error immediately instead of waiting.

**Expected behavior:**
- Requests to Payment fail instantly with a clear error
- Order Service resources are freed quickly
- System remains partially functional — Inventory calls still work
- Reduced latency on failed requests (ms instead of timeout seconds)

### Phase 3: Circuit Breaker

**Scenario:** Payment Service is intermittently failing.

**Pattern:** Track failure rate of downstream calls. When failures exceed a threshold, "open" the circuit — stop calling the failing service entirely for a cooldown period. Periodically allow a test request ("half-open") to check if the service has recovered.

**States:**
- **Closed** — Normal operation, requests pass through
- **Open** — Failures exceeded threshold, requests short-circuited with fallback response
- **Half-Open** — After cooldown, allow one test request to determine recovery

**Expected behavior:**
- Initial failures detected and circuit opens
- Subsequent requests return fallback instantly (no waiting)
- System auto-recovers when Payment Service comes back online
- Inventory Service completely unaffected throughout

### Phase 4: Bulkhead

**Scenario:** Payment Service hangs (slow responses, not crashing).

**Pattern:** Isolate downstream service calls into separate goroutine pools with bounded concurrency. Payment and Inventory each get their own pool — if one pool is exhausted, the other continues operating normally.

**Expected behavior:**
- Payment goroutine pool fills up with hanging requests
- New payment requests are rejected immediately (pool full)
- Inventory goroutine pool operates independently at full capacity
- System degrades gracefully — inventory-only operations are unaffected

## Fault Injection

Payment Service exposes admin endpoints to simulate failures:

| Endpoint | Effect |
|---|---|
| `POST /admin/crash` | Service stops responding (process exit) |
| `POST /admin/hang` | Responses delayed by 30s+ |
| `POST /admin/flaky?rate=0.5` | 50% of requests fail randomly |
| `POST /admin/recover` | Restore normal operation |

## Metrics & Evaluation

Load testing performed with **Locust** against each phase under identical conditions.

### Test Configuration

- **Users:** 100 concurrent
- **Spawn rate:** 10 users/sec
- **Duration:** 60 seconds per phase
- **Endpoints tested:** `POST /orders` (triggers both Payment and Inventory calls)

### Metrics Collected

| Metric | Description |
|---|---|
| **Latency (p50, p95, p99)** | Response time distribution |
| **Throughput (req/s)** | Successful requests per second |
| **Error Rate (%)** | Percentage of failed requests |
| **Availability** | Inventory Service availability during Payment failure |

### Expected Results Summary

| Phase | Latency (p95) | Error Rate | Inventory Available | Throughput |
|---|---|---|---|---|
| 1 - No Resilience | 30s+ (timeout) | ~100% | ❌ No | Near 0 |
| 2 - Fail Fast | < 100ms | ~50% (payment only) | ✅ Yes | Moderate |
| 3 - Circuit Breaker | < 50ms | ~50% → decreasing | ✅ Yes | High |
| 4 - Bulkhead | < 50ms | ~50% (payment only) | ✅ Yes | High |

## Quick Start

```bash
# Build all services
make build

# Run locally (4 terminals)
./bin/gateway
./bin/order
./bin/payment
./bin/inventory

# Run load test
locust -f locustfile.py --host=http://localhost:8080

# Trigger failure
curl -X POST http://localhost:8082/admin/crash

# Run full experiment (all phases)
./scripts/run_tests.sh
```

## AWS Deployment

```bash
# Deploy to EC2
./scripts/deploy.sh

# Run remote tests
locust -f locustfile.py --host=http://<ec2-public-ip>:8080
```
