# Homework 2 - MapReduce, Terraform, Docker... and Claude Code

## Part II: Terraform EC2 Infrastructure

This part demonstrates Infrastructure as Code (IaC) using Terraform to automate EC2 deployment on AWS.

### Overview

Instead of manually clicking through AWS Console, we use Terraform to define and deploy cloud infrastructure with code. This enables reproducibility and version control of infrastructure.

### Architecture

```
terraform.tfvars (secrets - gitignored)
      │
      ├──▶ ssh_cidr ──────────────────┐
      │                               │
      └──▶ ssh_key_name ──┐           │
                          │           ▼
┌─────────────────┐       │    ┌──────────────────────┐
│  data.aws_ami   │       │    │  aws_security_group  │
│  (AL2023 AMI)   │       │    │  (SSH from your IP)  │
└────────┬────────┘       │    └───────────┬──────────┘
         │                │                │
         │    ┌───────────┘                │
         │    │                            │
         ▼    ▼                            ▼
      ┌─────────────────────────────────────┐
      │          aws_instance               │
      │     (t2.micro EC2 instance)         │
      └──────────────────┬──────────────────┘
                         │
                         ▼
                  ┌─────────────┐
                  │   output    │
                  │ (public DNS)│
                  └─────────────┘
```

### Files

| File | Description |
|------|-------------|
| `main.tf` | Terraform configuration defining EC2, security group, and AMI lookup |
| `terraform.tfvars` | Variables (IP address, key pair name) - **not committed** |
| `.terraform.lock.hcl` | Provider version lock file (committed) |
| `.gitignore` | Excludes sensitive files from version control |

### Resources Created

| Resource | Purpose |
|----------|---------|
| `aws_security_group.ssh` | Allows SSH (port 22) only from my IP |
| `aws_instance.demo-instance` | t2.micro EC2 instance running Amazon Linux 2023 |

### Usage

#### Prerequisites
- Terraform installed
- AWS CLI installed
- AWS Academy Lab credentials
- Existing AWS key pair

#### Setup & Deploy

```bash
# 1. Configure AWS credentials
aws configure
aws configure set aws_session_token <YOUR_SESSION_TOKEN>

# 2. Create terraform.tfvars with your values
ssh_cidr     = "YOUR_PUBLIC_IP/32"
ssh_key_name = "your-key-name"

# 3. Initialize and deploy
terraform init
terraform apply -auto-approve

# 4. SSH into instance
ssh -i <PATH-TO-KEY.pem> ec2-user@<EC2-PUBLIC-DNS>

# 5. Clean up when done
terraform destroy -auto-approve
```

### Troubleshooting on macOS (Apple Silicon: M1/M2/M3)

#### Problem

On Apple Silicon Macs, Terraform may fail with:

```
Failed to load plugin schemas
timeout while waiting for plugin to start
```

or Terraform may hang during apply/destroy.

#### Root Cause

This happens when:
- Terraform is installed as **x86_64 (darwin_amd64)** using Intel Homebrew (`/usr/local`)
- But the machine is **arm64**
- Result: AWS provider is also downloaded as `darwin_amd64`, which runs unstably under Rosetta and causes plugin startup timeouts, high CPU usage, and hanging apply/destroy

#### Correct Fix (Permanent)

Use **native arm64 Terraform** from Apple Silicon Homebrew:

```bash
# Ensure arm64 Homebrew is used
eval "$(/opt/homebrew/bin/brew shellenv)"

# Install arm64 Terraform
brew install terraform

# Verify architecture
terraform version
# Must show: darwin_arm64
```

Then in the project:

```bash
rm -rf .terraform .terraform.lock.hcl
terraform init
```

Verify provider architecture:

```bash
file .terraform/providers/.../terraform-provider-aws_*
# Must show: arm64
```

After this, `terraform apply` and `terraform destroy` work reliably.

#### Quick Fix (if above doesn't work)

Remove macOS quarantine attribute:

```bash
xattr -dr com.apple.quarantine .terraform
xattr -dr com.apple.quarantine ~/.terraform.d 2>/dev/null || true
rm -rf .terraform .terraform.lock.hcl
terraform init
terraform apply -auto-approve
```

### Notes

- The EC2 instance will receive a new public IP/DNS each time it is recreated
- AWS Academy credentials expire periodically and must be reconfigured
- Update `ssh_cidr` if your IP address changes (e.g., switching networks)

---

## Part III: Docker Containerization

### Overview

Containerized the Go web service from Homework 1 using Docker and deployed it to AWS EC2.

### Dockerfile (Multi-stage Build)

```dockerfile
# ---- build stage ----
FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .

# ---- run stage ----
FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/server /server
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
```

### Why Multi-stage Build?

- **Smaller image**: Final image only contains the compiled binary, not the Go compiler
- **More secure**: Distroless image has no shell, reducing attack surface
- **Non-root user**: Runs as `nonroot` for better security

### Local Testing

```bash
cd hw1a
docker build -t cs6650-hw2-go .
docker run --rm -p 8080:8080 cs6650-hw2-go

# Test in another terminal
curl http://localhost:8080/albums
```

### EC2 Deployment

```bash
# SSH into EC2
ssh -i <key.pem> ec2-user@<EC2-DNS>

# Install dependencies
sudo dnf update -y
sudo dnf install -y git docker
sudo systemctl enable --now docker
sudo usermod -aG docker ec2-user

# Add swap (prevents build OOM on t2.micro)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Re-login, then clone and run
git clone https://github.com/YaoyiW27/CS6650.git
cd CS6650/hw1a
docker build -t cs6650-hw2-go .
docker run -d --name cs6650 -p 8080:8080 cs6650-hw2-go
```

### Screenshots

- `part3-docker-running-local.png`: Docker container running locally
- `part3-docker-running-ec2.png`: Docker container running on EC2 with curl test

---

## Part IV: Multi-Instance Data Inconsistency

### Overview

Deployed the same Dockerized Go service on two separate EC2 instances to demonstrate data inconsistency issues in distributed systems.

### Experiment

1. Created 2 EC2 instances using Terraform (`count = 2`)
2. Deployed identical Docker containers on both
3. Sent a POST request to only Instance B to add a new album
4. Compared GET responses from both instances

### Results

- **Instance A**: Returns original 3 albums
- **Instance B**: Returns 4 albums (including the new one)

### Analysis

I created two EC2 instances and ran the same Dockerized Go album service on both. At first, both instances returned the same album list. Then the test script sent a POST request to Instance 2 to add a new album. After that, Instance 2 showed the new album, but Instance 1 did not change and still returned the original list.

This happens because each EC2 instance runs an independent container with its own in-memory state. There is no shared database or synchronization between the two instances, so updates made to one instance do not propagate to the other.

In real deployments, we would use shared storage (e.g., a database/Redis) or place instances behind a load balancer with a shared backend to keep state consistent.

### Screenshots

- `part4-ec2-a.png`: Instance A response
- `part4-ec2-b.png`: Instance B response
- `part4-instanceA_vs_instanceB.png`: Side-by-side comparison
- `part4-test-result1.png`: Initial test results
- `part4-test-result2.png`: Results after POST to Instance B

---

## Part V: Bug Investigation with Claude Code

### Overview

Used Claude Code to analyze a mystery Lambda application and identify a concurrency bug by examining the source code and CloudWatch logs.

### Time Spent

Approximately **40 minutes** interacting with Claude Code (analysis took 1m 28s).

### Bug Found: Race Condition

**Location**: `postAlbumCount` function in `main.go` (lines 129-130)

```go
current := albumCounts[index].Count
albumCounts[index].Count = current + 1
```

**Problem**: This code spawns 10,000 goroutines that all try to increment the same counter **without synchronization**. The read-modify-write operation is not atomic, causing lost updates.

### Evidence from CloudWatch Logs

| Request | Final Count | Expected | Lost Increments |
|---------|-------------|----------|-----------------|
| 1 | 10000 | 10000 | 0 ✓ |
| 2 | 19994 | 20000 | 6 ✗ |
| 3 | 29987 | 30000 | 13 ✗ |
| 4 | 39987 | 40000 | 13 ✗ |
| 5 | 49974 | 50000 | 26 ✗ |
| 6 | 59974 | 60000 | 26 ✗ |
| 7 | 69974 | 70000 | 26 ✗ |
| 8 | 79973 | 80000 | 27 ✗ |
| 9 | 89958 | 90000 | 42 ✗ |

### The Fix

The fix would require synchronization using either:
- `sync.Mutex` to protect the counter
- `atomic.AddInt64()` for atomic increments
- A channel-based approach

### Files

- `2026-01-26-where-is-the-bug-check-maingo-and-then-cloudwat.txt`: Claude Code conversation export