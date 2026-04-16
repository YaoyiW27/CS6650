# Lessons Learned — CS6650 Final Project
## Mastodon Scaling Study
**Yaoyi Wang**

---

## What I Learned

**Bottlenecks are not where you expect them.**
Before this project, I assumed database saturation would be the first sign of trouble in a social network. The opposite was true. Across every experiment, PostgreSQL stayed under 16% CPU while the web layer (Puma) saturated first. Redis absorbed 84% of reads, and PostgreSQL never received meaningful direct load. This connected directly to the CAP theorem and queueing theory from class: the system was designed to protect the relational store by layering caches and rate limiters in front of it. Understanding this layered architecture changed how I think about performance — you have to trace the full request path before you can locate the real bottleneck.

**Eventual consistency is not just a theoretical concept.**
In HW10, we implemented leader-follower and leaderless replication and discussed how consistency guarantees differ across models. The Mastodon federation experiment made this concrete in a way no homework problem could. When Instance B was under 20-user load, a post from Instance A took 28.73 seconds to appear on Instance B's home timeline — and at 50 users, delivery failed entirely. The ActivityPub protocol is eventually consistent by design: posts will be delivered, but there is no guarantee of when. Loading the receiver delays "eventually" by seconds or indefinitely. One instance's load problem becomes another instance's consistency problem. This is a form of cascading degradation unique to federated architectures that the CAP and FLP results predict but don't make visceral until you measure it.

**Silent failures are more dangerous than hard failures.**
When I stopped the Sidekiq container in Experiment 4, HTTP latency actually improved — from 530ms to 89ms. From Locust's perspective, the system looked healthier than normal. Meanwhile, 2,680+ jobs were silently queuing up: federation deliveries never sent, notifications never received, followers never updated. This connects to the course discussion on asynchronous systems and observability: decoupling background work from the request path (the Sidekiq pattern) is essential for performance, but it creates failure modes that are completely invisible to standard HTTP monitoring. Queue depth alerting is not optional in production — it is the only way to detect this class of failure.

---

## What Went Wrong and Why

**The CloudFormation/ECS path cost a full week.**
I spent Week 1 iterating through five CloudFormation template versions, each failing due to a different AWS Academy Learner Lab IAM restriction. I should have tested for IAM permission limits on a minimal stack first before building the full deployment. The lesson is: validate your infrastructure assumptions before investing in a complex setup. In future projects, I would run a 10-minute IAM permission probe before committing to a deployment architecture.

**The combined federation test silently produced invalid data (Exp 3C Phase 1).**
When Yehe ran the combined sender-load federation test, Locust failed silently because the locust package was not installed in the active Python environment. The output was redirected to /dev/null, so there was no error visible. The first full dataset appeared valid but had no actual load applied. This taught me: never suppress error output from background processes, and always verify that infrastructure (load generators, monitoring scripts) is actually running before treating results as valid. This mirrors what the course covered about fault detection — a system that fails silently is harder to operate than one that fails loudly.

---

## How I Would Handle It Next Time

- Run a 10-minute IAM permission probe before committing to any cloud deployment architecture
- Redirect all subprocess output to log files, never /dev/null, during automated experiments
- Instrument queue depth from the start — not as an afterthought after observing failures
- Separate Sidekiq onto dedicated infrastructure to eliminate CPU contention with the web layer, which was the root cause of federation latency degradation in Exp 3B
- Use connection pooling (PgBouncer) from the beginning to prevent the PostgreSQL connection leak discovered in Exp 3C

---

## Course Concepts That Applied

This project touched more course concepts than any homework assignment:

- **CAP theorem**: Mastodon's ActivityPub delivery is AP — available and partition-tolerant, with eventual consistency. The federation latency experiment measured exactly how "eventual" that consistency becomes under load.
- **Queueing theory**: Sidekiq queue depth growing faster than it drains (Exp 4C at u=50) is a direct example of arrival rate exceeding service rate — Little's Law in action.
- **Asynchronous systems**: The decoupling of HTTP request handling from background job processing is the central architectural decision in Mastodon, and understanding it was the key to interpreting every experiment result.
- **Fault detection and observability**: Sidekiq's silent failure mode showed that correct behavior in distributed systems requires explicit monitoring of every async layer, not just the synchronous request path.
- **Vertical vs. horizontal scaling**: The t3.medium vs. t3.large comparison (Exp 2) showed that the same tuning that eliminates failures on one instance actively harms another — a direct demonstration of why scaling decisions cannot be made without knowing hardware constraints.
