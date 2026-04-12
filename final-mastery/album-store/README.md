# Album Store — ChaosArena v1

**CS6650 Distributed Systems — Final Mastery, Spring 2026**

A REST API service for managing photo albums with asynchronous photo processing, built for the ChaosArena automated scoring platform.

**Final Score: 162 / 190** (Correctness: 110/110 · Load: 52/80)

---

## Architecture

```
                         ┌──────────────────┐
    Client Request ────> │   Go + Gin       │
                         │   (EC2 t3.med)   │
                         └──┬─────────┬─────┘
                            │         │
                   sync     │         │  async (goroutine)
                            │         │
                  ┌─────────▼──┐   ┌──▼──────────┐
                  │ PostgreSQL │   │     S3       │
                  │ (RDS)      │   │ (photo files)│
                  └────────────┘   └──────────────┘
```

**Request flow:**

1. Client sends request to Go/Gin server on EC2
2. Album CRUD operations go directly to PostgreSQL (RDS)
3. Photo uploads: handler assigns `seq` synchronously via atomic DB counter, saves metadata to PostgreSQL, returns `202 Accepted` immediately
4. A background goroutine streams the photo to S3, then updates the DB status to `completed`
5. Photo deletes remove both the DB record and the S3 object

---

## Tech Stack

| Component        | Technology                |
|------------------|---------------------------|
| Language         | Go 1.25                   |
| HTTP Framework   | Gin                       |
| Database         | PostgreSQL 16 (AWS RDS)   |
| File Storage     | AWS S3                    |
| Compute          | AWS EC2 (t3.medium)       |
| Infrastructure   | Terraform                 |
| Local Dev        | Docker Compose            |

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `PUT` | `/albums/:album_id` | Create or update album (idempotent) |
| `GET` | `/albums/:album_id` | Get album by ID |
| `GET` | `/albums` | List all albums |
| `POST` | `/albums/:album_id/photos` | Upload photo (async, returns 202) |
| `GET` | `/albums/:album_id/photos/:photo_id` | Get photo status |
| `DELETE` | `/albums/:album_id/photos/:photo_id` | Delete photo |

---

## Key Design Decisions

### Synchronous `seq` Assignment

The `seq` number is assigned atomically in the POST handler using PostgreSQL's `INSERT ... ON CONFLICT DO UPDATE ... RETURNING`:

```sql
INSERT INTO album_seq (album_id, next_seq)
VALUES ($1, 2)
ON CONFLICT (album_id) DO UPDATE
SET next_seq = album_seq.next_seq + 1
RETURNING next_seq - 1
```

This guarantees monotonically increasing, unique sequence numbers per album even under concurrent uploads — no application-level locks needed.

### Async Photo Processing with Goroutines

Initially used SQS + a background worker polling loop, but switched to direct goroutine-based S3 uploads for lower latency. The handler streams the upload to a temp file on disk, returns 202 immediately, and a goroutine reads from the temp file to upload to S3. This avoids holding the full file in memory (critical for S15 large payloads on t3.medium with 4GB RAM).

### Album UPSERT with Correct Status Codes

Used PostgreSQL's `ON CONFLICT DO UPDATE` for idempotent PUT, and the `xmax` system column to distinguish insert (201) vs update (200) without extra queries.

---

## Database Schema

```sql
-- Albums
CREATE TABLE albums (
    album_id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    owner TEXT NOT NULL
);

-- Photos
CREATE TABLE photos (
    photo_id TEXT PRIMARY KEY,
    album_id TEXT NOT NULL REFERENCES albums(album_id),
    seq INT NOT NULL,
    status TEXT NOT NULL DEFAULT 'processing',
    url TEXT,
    UNIQUE(album_id, seq)
);

-- Per-album sequence counter
CREATE TABLE album_seq (
    album_id TEXT PRIMARY KEY REFERENCES albums(album_id),
    next_seq INT NOT NULL DEFAULT 1
);
```

---

## Local Development

```bash
# start PostgreSQL
docker-compose up -d

# run server
go run main.go

# test
curl http://localhost:8080/health
curl -X PUT http://localhost:8080/albums/test-1 \
  -H "Content-Type: application/json" \
  -d '{"album_id":"test-1","title":"Test","description":"desc","owner":"me@test.com"}'
curl -X POST http://localhost:8080/albums/test-1/photos -F "photo=@any-file.jpg"
```

---

## AWS Deployment

```bash
# provision infrastructure
cd terraform
terraform init
terraform apply

# cross-compile and deploy
GOOS=linux GOARCH=amd64 go build -o album-store-linux .
scp -i key.pem album-store-linux ubuntu@<EC2_IP>:~/album-store

# on EC2
export PORT=8080
export DATABASE_URL="postgres://...?sslmode=require"
export AWS_REGION="us-west-2"
export S3_BUCKET="album-store-photos-..."
export GIN_MODE=release
./album-store
```

---

## Score Progression

| Submission | Score | Key Change |
|------------|-------|------------|
| 1st | 149 | Initial deploy: SQS + worker + photo_data in DB |
| 2nd | 160 | Switched to goroutine + direct S3 upload |
| 3rd | 162 | Temp file streaming + EC2 upgrade to t3.medium |

**Correctness (S1–S10): 110/110** — all scenarios passed on first submission.

**Load tests:** S11 (15/15), S12 (7/15), S13 (15/15), S14 (15/15), S15 (0/20).

S15 bottleneck: large file uploads (~200MB) take 30+ seconds just for network transfer to EC2, causing high accept latency. S12 bottleneck: S3 PutObject latency under concurrent uploads.

---

## What I Would Improve

- **S3 multipart upload** for large files to parallelize the transfer
- **S3 Transfer Acceleration** to reduce upload latency
- **Connection pooling tuning** for higher concurrency
- Add **DB indexes** on `photos(album_id)` for faster queries
- Move to **ECS Fargate** with ALB for horizontal scaling
