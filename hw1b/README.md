# CS6650 Homework 1b - AWS EC2 Deployment & Load Testing

## Overview
This assignment deploys a Go-Gin REST API to AWS EC2 and performs load testing to analyze response time characteristics.

## Part 1: EC2 Deployment

### Steps Completed
1. Created EC2 instance (Amazon Linux 2023, t2.micro)
2. Configured Security Group (SSH port 22, HTTP port 8080)
3. Cross-compiled Go code for Linux: `GOOS=linux GOARCH=amd64 go build -o newmain main.go`
4. Uploaded binary to EC2 using `scp`
5. Successfully ran server and tested with `curl`

### How to Run
```bash
# SSH into EC2
ssh -i cs6650-key.pem ec2-user@<EC2_PUBLIC_IP>

# Run server
cd myapp
./newmain

# Test from local machine
curl http://<EC2_PUBLIC_IP>:8080/albums
```

## Part 2: Key Concepts (See Slides)

- Virtual Machines vs Local execution
- SSH & Security Groups
- Elastic IP addresses
- EC2 Instance Types
- GCP vs AWS comparison

## Part 3: Load Testing

### How to Run
```bash
cd hw1b
python3 load_test.py
```

### Results
| Metric | Value |
|--------|-------|
| Total Requests | 830 |
| Average Response Time | 36.10ms |
| Median Response Time | 35.55ms |
| 95th Percentile | 42.13ms |
| 99th Percentile | 53.18ms |
| Max Response Time | 98.51ms |

### Observations & Analysis

**1. Distribution Shape: Does your histogram show a long tail?**

Yes. Most requests cluster around 30-40ms, with a tail extending to 98ms. About 5% of requests exceed 42ms (95th percentile), and 1% exceed 53ms (99th percentile).

**2. Consistency: Are response times consistent or do you see patterns?**

Mostly consistent. The scatter plot shows stable times around 35-40ms with random spikes (not clustered), indicating sporadic network/system events rather than systematic issues.

**3. Percentiles: What's the difference between median and 95th percentile?**

- Median: 35.55ms
- 95th percentile: 42.13ms
- Difference: 6.58ms (~18% higher)

The gap is relatively small, but max (98.51ms) shows outliers can be 3x slower than typical.

**4. Infrastructure Impact: How does t2.micro contribute to variability?**

- Shared hardware with other AWS customers
- Burstable CPU with credit limits
- Limited resources (1 vCPU, 1GB RAM)
- "Low to moderate" network bandwidth

A dedicated larger instance would likely show more consistent performance.

**5. Scaling Implications: What if 100 concurrent users?**

Response times would increase significantly. The single vCPU would be overwhelmed, requests would queue up, and latency could jump to hundreds of milliseconds. Solutions: larger instance, multiple instances with load balancer, or auto-scaling.

**6. Network vs. Processing: What causes longer response times?**

Mostly network latency. The server does minimal processing (just returns JSON from memory). The ~36ms average matches typical cross-country round-trip time. To verify: run `curl localhost:8080/albums` from within EC2 — if <5ms, network is the main factor.