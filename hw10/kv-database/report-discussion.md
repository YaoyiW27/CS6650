# Distributed KV Database — Results Discussion

## 1. How the Load Test Generator Works

The load tester uses a **small key pool** (10 keys) to guarantee temporal locality — with only 10 distinct keys and 2000 requests, reads and writes to the same key are naturally clustered closely in time. Each request randomly selects a key from this pool and decides whether to read or write based on the configured write percentage. 20 concurrent goroutines execute the request plan in parallel. The client tracks the latest known version per key from write responses, allowing it to detect stale reads when a subsequent read returns an older version.

## 2. Latency Analysis

### Read Latency

The read latency results split into two distinct groups determined by the R parameter:

**R=1 configurations (Leader W=5/R=1 and Leaderless)** deliver reads in ~1–3ms median because they only read from one node's local memory. The distributions show a right-skewed shape with a long tail extending to ~10ms, likely caused by Go's garbage collector pauses and Docker networking jitter.

**R=3 and R=5 configurations (Leader W=3/R=3 and Leader W=1/R=5)** have read latencies of ~53–58ms median. This is dominated by the 50ms artificial sleep each follower applies before responding to internal read requests. The W=3/R=3 configuration shows slightly higher p99 values (~108ms at 1% writes) due to needing 3 responses — if one node is slow, the quorum must still wait. The W=1/R=5 configuration shows occasional outliers above 100ms when all 5 nodes must respond.

**Key insight:** Read latency is almost entirely determined by R, not by write volume. Even at 90% writes, read latencies remain stable because reads are independent operations that don't contend with the write path.

### Write Latency

Write latency shows an even more dramatic split:

**W=1 (Leader W=1/R=5)** achieves sub-millisecond writes (~0.5–3ms median) because the leader only updates its local store and returns immediately, replicating asynchronously. This is 800x faster than other configurations.

**W=3, W=5, and Leaderless (W=N)** all cluster around ~807–820ms median. This is the sum of the artificial delays: the leader sleeps 200ms between sending to each follower, and each follower sleeps 100ms before acknowledging. For W=5, the leader must send to 4 followers sequentially (4 × 200ms = 800ms) plus follower processing time. W=3 is similar because the leader still sends sequentially and must wait for 2 follower acks. The leaderless coordinator follows the same pattern.

**Long tail observation:** All high-W configurations show a tight distribution around 800–830ms with minimal tail, because the latency is dominated by the deterministic sleep values. The W=1 configuration shows more variance and a visible long tail (p99=10–28ms) because the fast path exposes underlying system jitter.

## 3. Stale Read Analysis

The stale read chart is the most revealing result:

**Only Leader W=1/R=5 produces stale reads**, and the rate increases dramatically with write volume: 0.1% at 1% writes → 2.9% at 10% → 17.7% at 50% → 35.7% at 90% writes.

This happens because W=1 returns to the client before followers are updated. When R=5, the quorum read contacts all nodes and picks the highest version — but if the async replication hasn't reached some followers yet, they return stale versions. The quorum read *should* still return the correct value (highest version wins), but the stale read detection in our client compares against the latest *client-known* version, which can race with in-flight async replications from concurrent writes.

**All other configurations show 0% stale reads** because:
- W=5/R=1: All nodes are confirmed updated before the write returns, so any single node read is guaranteed fresh.
- W=3/R=3: The quorum intersection property (W + R > N, i.e. 3 + 3 > 5) guarantees that at least one node in every read quorum has the latest write.
- Leaderless W=5/R=1: Same as Leader W=5/R=1 — all nodes confirmed before return.

## 4. Read-Write Interval Analysis

The read-write interval graphs show the time gap between a write to a key and the subsequent read of that same key. This is important because it reveals how our test generator creates opportunities for stale reads.

At **1% writes / 99% reads**, intervals are large (median 85–1100ms) because writes are rare, so reads often happen long after the last write — well outside any inconsistency window. At **90% writes / 10% reads**, intervals shrink dramatically (median 1.7–5.9ms) because frequent writes mean any read is likely very close in time to a recent write.

The **bimodal distributions** visible in W=5/R=1 and Leaderless graphs (spikes at both ~5ms and ~800ms) occur because slow writes (~800ms) create a "wall" — reads arriving during a write must wait, creating a cluster at the write duration boundary.

## 5. Which Configuration is Best for What?

| Configuration | Best For | Why |
|---|---|---|
| **Leader W=5, R=1** | Read-heavy workloads requiring strong consistency (e.g., user profile lookups, configuration stores) | Ultra-fast reads (~2ms), zero stale reads, but writes are slow (~810ms). Ideal when writes are rare. |
| **Leader W=1, R=5** | Write-heavy workloads tolerating eventual consistency (e.g., logging, analytics ingestion, event streams) | Sub-millisecond writes, but reads are slow (~55ms) and stale reads occur. Only use if reading stale data is acceptable. |
| **Leader W=3, R=3** | Balanced workloads needing consistency guarantees (e.g., e-commerce inventory, banking) | Quorum guarantees zero stale reads while offering moderate performance for both reads and writes. The "safe default" choice. |
| **Leaderless W=5, R=1** | High-availability read-heavy workloads (e.g., DNS, CDN origin) | Similar performance to Leader W=5/R=1 but no single point of failure. Any node can accept writes, improving availability if one node goes down. |

## 6. Summary

Our results clearly demonstrate the fundamental CAP theorem trade-off: you cannot simultaneously optimize for fast writes, fast reads, and strong consistency. Each N/R/W configuration makes a different trade-off along this spectrum, and the "right" choice depends entirely on the application's requirements.
