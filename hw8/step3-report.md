# STEP III: Database Comparison & Analysis Report

**Data Source**: `combined_results.json` (300 operations: 150 MySQL + 150 DynamoDB)

---

## Part 1: Performance Comparison Table

### Overall Metrics

| Metric | MySQL | DynamoDB | Winner | Margin |
|---|---|---|---|---|
| Avg Response Time (ms) | 38.9 | 42.4 | MySQL | 3.5ms |
| P50 Response Time (ms) | 37.6 | 40.5 | MySQL | 2.9ms |
| P95 Response Time (ms) | 54.0 | 48.1 | DynamoDB | 5.9ms |
| P99 Response Time (ms) | 66.4 | 168.6 | MySQL | 102.2ms |
| Success Rate (%) | 100% | 100% | Tie | 0% |
| Total Operations | 150 | 150 | | |

### Operation-Specific Breakdown

| Operation | MySQL Avg (ms) | DynamoDB Avg (ms) | Faster By |
|---|---|---|---|
| CREATE_CART | 41.5 | 40.8 | DynamoDB by 0.7ms |
| ADD_ITEMS | 37.6 | 47.0 | MySQL by 9.4ms |
| GET_CART | 37.5 | 39.4 | MySQL by 1.9ms |

**Key Observations**:
- CREATE_CART is nearly identical — both databases handle simple inserts efficiently.
- ADD_ITEMS shows the biggest gap: MySQL's `INSERT ... ON DUPLICATE KEY UPDATE` is a single atomic operation, while DynamoDB required a Scan → read items → UpdateItem flow (3 round trips).
- GET_CART is close, but MySQL's indexed query is slightly faster than DynamoDB's Scan-based lookup by `numeric_id`.
- **Interesting P95 result**: DynamoDB actually wins at P95 (48.1ms vs 54.0ms), showing more consistent performance for the majority of requests. However, its P99 (168.6ms) is much worse than MySQL (66.4ms), indicating occasional cold-start or capacity allocation spikes.

### Consistency Model Impact Assessment

**MySQL (ACID)**:
- Every read returns the most recently committed data. After creating a cart and immediately reading it, the data was always present and consistent.
- Transactions guarantee that cart + item operations are atomic.

**DynamoDB (Eventual Consistency)**:
- During testing with 150 sequential operations, I did not observe any consistency delays. Read-after-write returned correct data 100% of the time.
- DynamoDB offers strong consistency as an option (ConsistentRead=true), but default eventually consistent reads were sufficient for this workload.
- At higher concurrency or multi-region setups, eventual consistency would become more visible.

---

## Part 2: Resource Efficiency Analysis

### Resource Utilization Comparison

**MySQL (RDS)**:
- Requires provisioning a `db.t3.micro` instance (~$15/month) regardless of traffic.
- Connection pooling configuration needed (I used 25 max connections, 10 idle).
- Fixed cost: instance runs 24/7 even with zero traffic.
- Operational overhead: monitoring connections, storage, backups.

**DynamoDB**:
- PAY_PER_REQUEST mode: zero cost at zero traffic, scales automatically.
- No connection pool management — SDK handles everything.
- No instance to manage, patch, or resize.
- Cost is purely usage-based: ~$1.25 per million write units, ~$0.25 per million read units.

### Scaling Analysis

- **MySQL**: Vertical scaling (upgrade instance) or read replicas. Requires downtime or manual intervention. Predictable costs but potential over-provisioning.
- **DynamoDB**: Automatic horizontal scaling with no downtime. Costs scale linearly with usage. Better for unpredictable traffic spikes.

---

## Part 3: Real-World Scenario Recommendations

### Scenario A: Startup MVP
*100 users/day, 1 developer, limited budget, quick launch*

**Recommendation: DynamoDB**

**Key Evidence**: DynamoDB's PAY_PER_REQUEST mode means near-zero cost at low traffic. No database instance to manage means one developer can focus on features, not ops. My implementation required zero connection pool tuning. Setup was simpler — no security groups, subnet groups, or instance sizing decisions.

### Scenario B: Growing Business
*10K users/day, 5 developers, moderate budget, feature expansion*

**Recommendation: MySQL**

**Key Evidence**: As the product grows, the need for complex queries (customer history, analytics, joins across tables) favors SQL. My MySQL schema's `idx_customer_id` index directly supports customer history queries. ADD_ITEMS was 9.4ms faster with MySQL, and at 10K users/day these savings compound. The team of 5 likely has SQL experience, reducing learning curve.

### Scenario C: High-Traffic Events
*50K normal, 1M spike users, revenue-critical*

**Recommendation: DynamoDB**

**Key Evidence**: DynamoDB auto-scales with no configuration changes. My test showed consistent performance (39.4ms avg GET) even as data grew. MySQL's fixed instance would require pre-scaling before events and could still hit connection limits. DynamoDB's P99 spike (170ms) was a cold-start effect that wouldn't repeat under sustained load.

### Scenario D: Global Platform
*Millions of users, multi-region, 24/7 availability*

**Recommendation: DynamoDB (with MySQL for analytics)**

**Key Evidence**: DynamoDB Global Tables provide multi-region replication out of the box. MySQL multi-region requires custom replication setup. For the shopping cart use case specifically, DynamoDB's simple access patterns (create, get by ID, update items) are a perfect fit. Complex reporting and analytics can be offloaded to a separate MySQL/PostgreSQL instance.

---

## Part 4: Evidence-Based Architecture Recommendations

### 1. Shopping Cart Winner: MySQL (marginally)

For this specific test, MySQL wins overall with a 3.5ms average advantage and significantly better tail latency (P99: 65ms vs 170ms). The ADD_ITEMS operation was 9.4ms faster due to MySQL's efficient `ON DUPLICATE KEY UPDATE` vs DynamoDB's multi-step Scan+Update approach.

### 2. Supporting Evidence
- Response time advantage: 3.5ms overall, 9.4ms on ADD_ITEMS
- Implementation complexity: MySQL required more infrastructure (RDS instance, security groups, connection pooling) but simpler application code. DynamoDB had simpler infrastructure but more complex application logic (Scan workaround, reserved keyword handling, UUID-to-int mapping).
- Both achieved 100% success rate across 150 operations.

### 3. When to Choose DynamoDB Instead
- When traffic is unpredictable (auto-scaling is critical)
- When multi-region deployment is required
- When operational overhead must be minimized (no instances to manage)
- When cost must scale to zero during low-traffic periods
- If I had used `cart_id` (UUID) directly as the API identifier instead of mapping to int, DynamoDB would have been faster on GET and ADD_ITEMS (direct GetItem instead of Scan).

### 4. Polyglot Strategy for Complete E-commerce System
- **Shopping carts**: DynamoDB — session-based, simple access patterns, needs auto-scaling during sales events
- **User sessions**: DynamoDB — key-value lookups, TTL expiration, high write volume
- **Product catalog**: MySQL — complex queries (search, filter, sort by price/category), JOINs between products and categories
- **Order history**: MySQL — relational data (orders → line items → shipping), complex reporting queries, ACID transactions for payment processing

---

## Part 5: Learning Reflection

### What Surprised Me?
- MySQL and DynamoDB performance was much closer than expected. I anticipated DynamoDB would be significantly faster for simple key-value operations, but the network round-trip to RDS vs DynamoDB was the dominant factor, making raw database speed differences less impactful.
- DynamoDB's reserved keywords (like "items") caused a runtime error that was not caught at compile time — a sharp edge in the development experience.
- The DynamoDB implementation required more application-level complexity (UUID generation, Scan for numeric ID lookup, manual item list management) compared to MySQL's straightforward SQL.

### What Failed Initially?
- **Docker platform mismatch**: Built ARM image on M-series Mac, but ECS Fargate runs AMD64. Required explicit `--platform linux/amd64` flag.
- **RDS connectivity**: Initially couldn't connect from local machine because RDS was in a private subnet. Solved by deploying to ECS instead of testing locally.
- **DynamoDB reserved keyword**: `items` is reserved — had to use `ExpressionAttributeNames` with `#items` alias.
- **DynamoDB ID mapping**: The CartStore interface uses `int` IDs (natural for MySQL auto-increment), but DynamoDB's partition key is a UUID string. Had to implement a hash-based mapping, which required Scan operations instead of efficient GetItem. This was the main performance penalty.

### Key Insights Gained
- **Choose MySQL when**: You need complex queries, JOINs, transactions, or your team knows SQL well. The relational model is powerful for interconnected data.
- **Choose DynamoDB when**: You need auto-scaling, multi-region, or simple access patterns. Design your API around DynamoDB's strengths (use UUIDs as IDs, embed related data).
- **For another student**: Design your DynamoDB schema around your access patterns FIRST. Don't try to force a relational model onto DynamoDB — if I had used UUID strings as cart IDs in the API, the DynamoDB implementation would have been both simpler and faster.
- **Hands-on insight**: Reading about SQL vs NoSQL trade-offs is very different from experiencing them. The implementation complexity trade-off (simpler infrastructure vs simpler application code) only becomes clear when you build both.
