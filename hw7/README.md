# HW7: Synchronous vs Asynchronous Order Processing

## Overview

This project explores the trade-offs between synchronous, asynchronous (SNS/SQS), and serverless (Lambda) architectures for an e-commerce order processing system. A simulated 3-second payment verification bottleneck exposes how each architecture handles load.

## Architecture

### Part II: Sync vs Async (ECS + SNS/SQS)

```
Sync:   Client → ALB → order-receiver → 3s payment delay → 200 OK

Async:  Client → ALB → order-receiver → SNS → 202 Accepted
                                          ↓
                                     SQS Queue
                                          ↓
                                   order-processor (goroutine workers) → 3s delay
```

### Part III: Serverless (Lambda)

```
Client → ALB → order-receiver → SNS → Lambda (auto-scaled, 3s delay)
```

## Project Structure

```
hw7/
├── part2/
│   ├── order-receiver/        # API service (sync + async endpoints)
│   │   ├── main.go            # HTTP server, AWS session init
│   │   ├── handlers.go        # /health, /orders/sync, /orders/async
│   │   ├── models.go          # Order, Item, OrderResponse structs
│   │   ├── go.mod
│   │   └── Dockerfile
│   ├── order-processor/       # SQS consumer service
│   │   ├── main.go            # Entry point, configurable NUM_WORKERS
│   │   ├── worker.go          # SQS polling, worker pool, message processing
│   │   ├── models.go          # Order, Item, SNSMessage structs
│   │   ├── go.mod
│   │   └── Dockerfile
│   ├── locust/
│   │   └── locustfile.py      # Load test: OrderUser (sync) + AsyncOrderUser
│   ├── terraform/             # Infrastructure as Code
│   │   ├── main.tf            # AWS provider
│   │   ├── variables.tf       # Input variables (CIDRs, images, worker count)
│   │   ├── vpc.tf             # VPC, subnets, NAT Gateway, route tables
│   │   ├── alb.tf             # ALB, target group, security groups
│   │   ├── sns_sqs.tf         # SNS topic, SQS queue, subscription
│   │   ├── iam.tf             # LabRole reference
│   │   ├── ecs.tf             # ECS cluster, task definitions, services
│   │   ├── outputs.tf         # ALB DNS, SNS ARN, SQS URL
│   │   └── README.md          # Terraform-specific deployment guide
│   └── screenshots/           # All test evidence
├── part3/
│   ├── lambda/
│   │   ├── main.go            # Lambda handler (SNS trigger, 3s delay)
│   │   ├── go.mod
│   │   └── Makefile           # Build + zip for deployment
│   └── terraform/
│       ├── main.tf            # Lambda function, SNS subscription
│       ├── variables.tf
│       └── outputs.tf
└── README.md                  # This file
```

## Key Results

### Part II: Performance Comparison

| Test | RPS | Avg Response | Requests (60s) | Failures |
|------|-----|-------------|-----------------|----------|
| Sync — 5 users (normal) | 0.4 | 12,487ms | 17 | 0% |
| Sync — 20 users (flash sale) | 0.33 | 41,309ms | 31 | 0% |
| Async — 20 users, 1 worker | 59.3 | 36ms | 3,282 | 0% |
| Async — 20 users, 5 workers | 60.6 | 36ms | 3,861 | 0% |
| Async — 20 users, 20 workers | 59.3 | 34ms | 6,199 | 0% |
| Async — 20 users, 100 workers | 60 | 34ms | 4,834 | 0% |

**Key insight:** Async accepts **~150x more orders** than sync under flash sale load. API response time drops from 41 seconds to 36 milliseconds.

### Part II: Bottleneck Analysis

- Payment processor speed: 1 order per 3 seconds
- With 20 concurrent users, max throughput: **0.33 orders/second**
- Flash sale demand: ~60 orders/second
- Orders lost per second: ~59.67/second

### Part II: Worker Scaling Analysis

| Workers | Processing Rate | Queue Growth Rate | Prevents Buildup? |
|---------|----------------|-------------------|-------------------|
| 1 | 0.33/s | ~59.67/s | No |
| 5 | 1.67/s | ~58.33/s | No |
| 20 | 6.67/s | ~53.33/s | No |
| 100 | 33.33/s | ~26.67/s | No |
| **180** | **60/s** | **0/s** | **Yes (minimum)** |

Minimum workers to prevent queue buildup at 60 orders/second: **180** (60 orders/s × 3s per order).

### Part III: Lambda Observations

**Cold Start:**
- Init Duration: **69.85ms** (2.3% overhead on 3s processing)
- Occurs on first invocation and after ~5+ minutes idle
- Negligible impact for 3-second payment processing

**Cost Comparison (10,000 orders/month):**

| | ECS (Part II) | Lambda (Part III) |
|---|---|---|
| Monthly cost | $17 (2 tasks × $8.50) | **$0** (within free tier) |
| Break-even | Always $17 | Free until ~267K orders/month |
| Scaling | Manual (adjust workers) | Automatic |
| Queue management | Required (SQS) | Not needed |

**Lambda free tier covers:** 1M requests + 400K GB-seconds/month. For 3s/0.5GB orders: 400K ÷ 1.5 = ~267K orders FREE.

### Part III: Should the Startup Switch to Lambda?

Yes, for a startup processing under 267K orders/month, Lambda is the better choice. The cost advantage is compelling ($0 vs $17/month), cold start overhead is negligible (70ms on 3s processing), and the elimination of operational complexity (no queue monitoring, no worker scaling, no 3am alerts) frees the team to focus on product development. The trade-off is losing SQS's retry guarantees — SNS retries only twice before discarding — but for a startup prioritizing speed and cost efficiency, this is acceptable. As order volume grows beyond 267K/month or reliability requirements increase, migrating back to the ECS/SQS architecture provides a clear upgrade path.

## Analysis Questions

**How many times more orders did async accept vs sync?**
Async accepted ~106x more orders in 60 seconds (3,282 vs 31).

**What causes queue buildup and how to prevent it?**
Queue buildup occurs when the ingestion rate exceeds the processing rate. Prevention requires scaling workers until processing rate ≥ ingestion rate (180 workers for 60 orders/s at 3s each).

**When would you choose sync vs async in production?**
Use sync when immediate confirmation is required and processing is fast (<100ms). Use async when processing is slow, load is unpredictable, or you need to decouple services for resilience.

## Deployment

See [part2/terraform/README.md](part2/terraform/README.md) for detailed deployment instructions.

## Cleanup

```bash
cd part2/terraform && terraform destroy
cd part3/terraform && terraform destroy
aws ecr delete-repository --repository-name order-receiver --force --region us-west-2
aws ecr delete-repository --repository-name order-processor --force --region us-west-2
```