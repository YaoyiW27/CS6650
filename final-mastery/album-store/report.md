# CS6650 Final Mastery — Report

**Yaoyi Wang · wang.yaoyi@northeastern.edu**
**Spring 2026**

---

## 1. How many submissions before passing all critical scenarios, and most common failure?

It took 1 submission to pass all critical scenarios (S1–S5). They all passed on the first try. The most common failure across later submissions was S15 (large payload upload) with 100% error rate, caused by `io.ReadAll` loading entire large files into memory and crashing the t3.micro instance (1GB RAM).

## 2. Where are your photo files stored, and why?

Photos are stored in **AWS S3**. S3 is the natural choice because it handles arbitrary file sizes, provides durable storage, and serves files via public URLs that ChaosArena can fetch directly. Storing files on EC2 local disk wouldn't survive instance restarts, and storing binary data in PostgreSQL would be slow and bloat the database.

## 3. Describe your deployment setup.

One EC2 instance (t3.medium) runs the Go/Gin HTTP server. It connects to an RDS PostgreSQL 16 instance for metadata (albums, photos, sequence counters) and to S3 for photo file storage. All resources are in us-west-2 and provisioned via Terraform. The EC2 instance uses an IAM instance profile (LabInstanceProfile) for S3 access.

## 4. Did you use a reverse proxy or load balancer?

No. The Go/Gin server listens directly on port 8080 on the EC2 instance. For a production system I would add an ALB in front for TLS termination and horizontal scaling, but for this assignment a single instance was sufficient.

## 5. How does your background worker get notified of new photos?

Initially I used AWS SQS with a polling worker, but switched to **Go goroutines** launched directly from the POST handler. When a photo upload arrives, the handler writes the file to a temp file on disk, returns 202 immediately, and spawns a goroutine that reads the temp file and uploads it to S3. This eliminated the SQS round-trip latency and improved S12/S14 scores significantly.

## 6. Why must `seq` be assigned in the POST handler, and how did you ensure correctness?

If `seq` were assigned in the background worker, concurrent uploads could receive their 202 responses before seq numbers are determined, making the response unreliable. I used PostgreSQL's atomic upsert to assign seq synchronously:

```sql
INSERT INTO album_seq (album_id, next_seq)
VALUES ($1, 2)
ON CONFLICT (album_id) DO UPDATE
SET next_seq = album_seq.next_seq + 1
RETURNING next_seq - 1
```

This is a single atomic SQL statement — PostgreSQL's row-level locking guarantees that concurrent uploads to the same album get unique, monotonically increasing seq numbers without any application-level locks.

## 7. What happens if the worker crashes halfway through processing?

The photo record stays in `status = 'processing'` in the database permanently. There is no retry mechanism — the photo will never reach `completed`. If the goroutine fails during S3 upload, it updates the status to `failed`. To improve this, I would add a cleanup job that marks stale `processing` photos as `failed` after a timeout.

## 8. What does your database schema look like?

Three tables:

- **albums** — `album_id` (PK), `title`, `description`, `owner`. Stores album metadata.
- **photos** — `photo_id` (PK), `album_id` (FK), `seq`, `status`, `url`. Has a `UNIQUE(album_id, seq)` constraint to prevent duplicate sequence numbers.
- **album_seq** — `album_id` (PK, FK), `next_seq`. A dedicated counter table for atomic seq assignment per album.

## 9. Did you add any indexes?

Only the implicit indexes from primary keys and the `UNIQUE(album_id, seq)` constraint. I did not add additional indexes. In hindsight, an index on `photos(album_id)` would help the photo status lookups and list queries under load.

## 10. Which load test scenario was hardest, and what bottleneck did you discover?

**S15 (Large Payload Upload)** was the hardest — I scored 0/20 across all three submissions. The bottleneck is twofold: (1) network transfer time for large files (~200MB) to the EC2 instance takes 30+ seconds, and (2) the initial implementation used `io.ReadAll` which loaded the entire file into memory, causing OOM on the t3.micro. Upgrading to t3.medium and switching to temp file streaming fixed the OOM but the network transfer latency remained.

## 11. What was the single most impactful change for load test scores?

Switching from **SQS + worker** to **direct goroutine S3 upload**. The original architecture stored photo binary data in a PostgreSQL `photo_data` table, sent a message to SQS, then a worker polled SQS, read the data from DB, and uploaded to S3. Removing this entire chain and uploading directly from a goroutine improved my score from 149 to 160 in one change. S12 went from 0 to 6 points, and S14 from 9 to 14.

## 12. How did you handle concurrent writes?

- **Album creates:** PostgreSQL `INSERT ... ON CONFLICT DO UPDATE` (upsert) is atomic and handles concurrent PUTs to the same album_id safely.
- **Photo seq assignment:** The `album_seq` table with `ON CONFLICT DO UPDATE ... RETURNING` guarantees unique seq numbers under concurrent uploads via row-level locking.
- **No application-level locks** — all concurrency control is delegated to PostgreSQL.

## 13. Describe a specific bug you diagnosed using logs.

On the first deploy to EC2, the service failed to start with: `failed to ping db: pq: no pg_hba.conf entry for host "172.31.40.135", user "postgres", no encryption`. The RDS instance required SSL connections, but my DATABASE_URL had `sslmode=disable`. Changing it to `sslmode=require` fixed the connection immediately. I diagnosed this directly from the Go server's startup error log.

## 14. How did you test locally before submitting?

I ran PostgreSQL locally via `docker-compose up -d`, then ran `go run main.go` and tested every endpoint with curl commands: health check, album create/get/list, photo upload/status/delete. I verified the full lifecycle — upload returns 202 with seq=1, status transitions from `processing` to `completed`, the file URL is accessible, and delete returns 204 with subsequent GET returning 404.

## 15. If you had another week, what would you change?

I would implement **S3 multipart upload** using the AWS SDK's upload manager to parallelize large file transfers, which should significantly improve S15 accept latency. I would also add a database index on `photos(album_id)` and experiment with increasing the DB connection pool size to improve S12 concurrent upload throughput.

## 16. How did you add value over and above what Claude could do?

Claude provided the initial code structure and architecture recommendations, but I made some debugging and optimization decisions myself. For example, after each ChaosArena submission I analyzed the per-scenario results to decide optimization priority — seeing S12 at 33% error rate and S15 at 100% error rate, I prioritized rewriting the S3 upload path over tuning database queries because that's where the most points were lost. Understanding *why* each change worked (not just *what* to change) came from my own distributed systems knowledge built throughout the course.
