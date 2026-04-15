# CS6650 Distributed Systems — Study Guide / 易错点整理

---

## 1. RPC vs REST

**RPC (Remote Procedure Call / 远程过程调用)** — Action-oriented, like calling a function on a remote server.
以**动作**为中心，像调用远程服务器上的函数。

```
POST /createUser        → create a user
POST /getUser           → get a user
POST /deleteUser        → delete a user
```

**REST (Representational State Transfer)** — Resource-oriented, use HTTP verbs on nouns.
以**资源**为中心，用 HTTP 动词操作名词。

```
POST   /users           → create a user
GET    /users/123       → get a user
DELETE /users/123       → delete a user
```

**Key difference / 核心区别：**

| | RPC | REST |
|---|---|---|
| Thinking / 思维方式 | Call a function / 调用函数 | Operate on resource / 操作资源 |
| URL style | Verb (`/getUser`) / 动词 | Noun (`/users/123`) / 名词 |
| Coupling / 耦合度 | High (must know function name) / 高 | Low (uniform GET/POST/PUT/DELETE) / 低 |
| Examples / 例子 | gRPC, Thrift | Your Album Store API / 你的作业 |

Your Album Store is REST: `PUT /albums/:id`, `GET /albums`, `POST /albums/:id/photos`.
你的 Album Store 就是 REST 风格。

---

## 2. Caching Patterns / 缓存模式

### Write-Behind (Write-Back) / 写后缓存
Write to cache → return OK immediately → cache syncs to DB asynchronously.
写 cache → 立刻返回 OK → cache 异步同步到 DB。

- Fast writes, but data loss risk if cache crashes before syncing.
- 写很快，但 cache 崩了数据会丢。
- Your photo upload is similar: return 202, goroutine writes S3 in background.
- 你的 photo upload 类似：返回 202，goroutine 后台写 S3。

### Write-Through / 写穿缓存
Write to cache AND DB synchronously → then return OK.
同时写 cache 和 DB，都完成了才返回 OK。

- Slow writes, but data is always consistent and safe.
- 写慢，但数据安全，cache 和 DB 始终一致。

### Cache-Aside (Lazy Loading) / 旁路缓存
Read: check cache → miss → read DB → put result back in cache.
读：先查 cache → 没有 → 查 DB → 把结果塞回 cache。

- App manages the cache itself. Most common Redis pattern.
- 应用自己管 cache，最常见的 Redis 用法。

### Read-Through / 读穿缓存
Read: app talks to cache only → cache loads from DB itself → returns data.
读：应用只跟 cache 说话 → cache 自己去查 DB → 返回给应用。

- Difference from Cache-Aside: the cache is responsible for loading, not the app.
- 和 Cache-Aside 区别：cache 自己负责加载，应用不需要知道 DB。

---

## 3. Resilience Patterns / 弹性模式

### Circuit Breaker / 熔断器
Like a fuse in your house. / 类比：家里的保险丝。

```
Closed (normal / 正常)
  → too many failures / 连续失败太多
Open (tripped, reject all / 跳闸，拒绝所有请求)
  → wait timeout / 等待超时
Half-Open (test one request / 试一个请求)
  → success → Closed / 成功 → 恢复
  → failure → Open  / 失败 → 继续跳闸
```

**Purpose / 目的：** Prevent a failed service from dragging down the whole system.
防止一个挂掉的服务拖垮整个系统。

### Bulkhead / 隔舱模式
Like watertight compartments on a ship. One floods, others stay dry.
类比：船的隔水舱。一个舱进水了，其他舱不受影响。

Isolate resources: Service A gets pool 1, Service B gets pool 2. A crashes → B still works.
隔离资源：服务 A 用线程池 1，服务 B 用线程池 2。A 挂了不影响 B。

**Purpose / 目的：** Fault isolation. One slow service can't eat all resources.
故障隔离，防止一个慢服务吃光所有资源。

### Fail Fast / 快速失败
If you know a request will fail, return error immediately. Don't waste time.
如果知道请求一定会失败，立刻返回错误。

- DB pool full → 503 immediately / 连接池满 → 直接 503
- Missing params → 400 immediately / 缺参数 → 直接 400
- Circuit breaker open → reject immediately / 熔断器跳闸 → 直接拒绝

### How they work together / 三者关系

```
Request → Fail Fast (bad params? reject)
        → Bulkhead (pool full? reject, protect others)
        → Circuit Breaker (downstream dead? reject)
        → Normal processing
```

---

## 4. Connection Pooling / 连接池

Creating a new DB connection per request is slow (TCP handshake + auth).
每次请求创建新连接很慢。

Pool: pre-create connections, borrow when needed, return when done.
连接池：预先创建连接，借用归还。

```go
conn.SetMaxOpenConns(50)                    // max 50 open / 最多 50 个
conn.SetMaxIdleConns(25)                    // keep 25 idle / 空闲保留 25
conn.SetConnMaxLifetime(30 * time.Minute)   // recycle after 30min / 30 分钟换
conn.SetConnMaxIdleTime(5 * time.Minute)    // close idle after 5min / 空闲 5 分钟关
```

Your `db.go` uses this. / 你的 `db.go` 就用了连接池。

---

## 5. Replication / 复制

### Leader-Follower (Master-Slave) / 主从复制
- One Leader handles all writes / 一个 Leader 处理写
- Followers replicate data, handle reads / Follower 复制数据，处理读
- Leader dies → elect new one (failover) / Leader 挂了 → 选新的

### Leaderless / 无主复制
- Any node can read/write / 任何节点都能读写
- Write to W nodes, read from R nodes / 写 W 个，读 R 个
- W + R > N → guaranteed to read latest / 保证读到最新
- Examples: DynamoDB, Cassandra

Your HW10: both Leader-Follower and Leaderless in Go.
你 HW10 用 Go 实现了两种。

---

## 6. Lamport Clock vs Vector Clock / 逻辑时钟

### Lamport Clock / Lamport 时钟
Each process has one **counter** / 每个进程一个计数器：
- Local event: counter++
- Send: counter++, attach to message / 发消息附上
- Receive: counter = max(local, received) + 1

**Can:** If A → B, then L(A) < L(B).
**Cannot:** L(A) < L(B) does NOT mean A → B. Could be concurrent.
L(A) < L(B) 不代表 A 在 B 之前，可能是并发。

### Vector Clock / 向量时钟
Each process has a **vector** [P1, P2, P3...] / 每个进程一个向量：
- Local event: own component++ / 自己的分量 +1
- Send: own++, attach full vector / 附整个向量
- Receive: max per component, own++ / 每个分量取 max

**Can:** Accurately detect causality AND concurrency.
可以准确判断因果和并发。

```
A = [1, 0, 0]    B = [0, 0, 1]
A[0] > B[0], but A[2] < B[2] → concurrent / 并发
```

| | Lamport | Vector |
|---|---|---|
| Structure / 结构 | One number / 一个数 | N numbers / N 个数 |
| Detect causality / 因果 | Partial / 部分 | Full / 完全 |
| Detect concurrency / 并发 | No / 不能 | Yes / 能 |
| Overhead / 开销 | Small / 小 | Large / 大 |

---

## 7. CAP Theorem / CAP 定理

At most 2 of 3 / 最多满足两个：

- **C (Consistency / 一致性)** — All nodes see same data / 所有节点同样数据
- **A (Availability / 可用性)** — Every request gets a response / 每个请求有响应
- **P (Partition Tolerance / 分区容忍)** — Works despite network splits / 网络分区时继续

P is mandatory, so real choice is CP vs AP / P 必须有，实际选 CP 还是 AP：

| | CP | AP |
|---|---|---|
| During partition / 分区时 | Reject to stay consistent / 拒绝保一致 | Serve but data may be stale / 服务但数据可能旧 |
| Examples | MySQL, ZooKeeper | Cassandra, DynamoDB |
| Your project | — | Mastodon (AP) |

---

## 8. Consensus / 共识算法

### Raft
- Leader election: candidate gets majority votes / 多数票当选 Leader
- Log replication: Leader sends to Followers, commit after majority ack / 多数确认后 commit
- Used by etcd, Kubernetes

### Paxos
- Same goal, more complex / 目的相同但更复杂
- Proposer → Acceptor → Learner
- Harder to implement / 难实现

**Core / 核心：** Majority agreement = consensus. >50% alive = system works.
超过半数活着就能达成共识。

---

## 9. Idempotency / 幂等性

Same operation, once or many times → same result.
同一操作执行一次和多次，结果一样。

```
Idempotent / 幂等:
  PUT /albums/123       → same result every time
  DELETE /photos/456    → already gone
  GET /albums/123       → always idempotent

NOT idempotent / 不幂等:
  POST /albums/123/photos → each call creates new photo (different seq)
```

Your `PUT /albums/:id` uses UPSERT = idempotent.
你的 PUT 用了 UPSERT = 幂等。

---

## 10. Async vs Sync / 异步 vs 同步

| | Sync / 同步 | Async / 异步 |
|---|---|---|
| Client waits / 等待 | Until done / 处理完才返回 | Immediate / 立刻返回 |
| Status code | 200 OK | 202 Accepted |
| Good for / 适合 | Fast ops / 快操作 | Slow ops (upload, email) / 慢操作 |
| Your project | Album CRUD | Photo upload |

---

## 11. Consistent Hashing / 一致性哈希

Problem: N servers, add/remove one → traditional hash (key % N) remaps almost everything.
问题：加减一台服务器，传统哈希几乎全部重新分配。

Solution: servers on a hash ring. Key maps to next server clockwise.
方案：服务器放哈希环上，key 顺时针找下一个。

- Add server → only neighboring keys move / 加服务器 → 只影响相邻 key
- Virtual nodes for even distribution / 虚拟节点让分布更均匀

---

## 12. MapReduce

Process large datasets in parallel across a cluster.
集群上并行处理大数据。

```
Input → Map (transform) → Shuffle (group by key) → Reduce (aggregate)

Word count example:
Map:     "hello world hello" → [("hello",1), ("world",1), ("hello",1)]
Shuffle: group by key       → {"hello": [1,1], "world": [1]}
Reduce:  sum                → {"hello": 2, "world": 1}
```

Your HW did MapReduce on AWS ECS.
你的 HW 在 ECS 上做过。
