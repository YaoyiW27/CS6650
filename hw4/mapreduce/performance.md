# HW4: Awesome Tools for Workflows

## Part II: ECR/ECS/Fargate Deployment

Successfully deployed the hello-service (Go REST API from HW1) to AWS using ECR, ECS, and Fargate. The process involved pushing a Docker image to ECR, creating an ECS cluster and task definition, and running a Fargate task with a public IP. Tested GET and POST endpoints successfully via curl.

## Part III: MapReduce Word Count with ECS and S3

### Architecture

Implemented a simplified MapReduce system using 5 ECS Fargate tasks and an S3 bucket to count word occurrences in Shakespeare's Hamlet (~159KB).

- **Splitter (1 task):** Reads the text file from S3, splits it into 3 equal-sized chunks, uploads them back to S3.
- **Mapper (3 tasks):** Each mapper reads one chunk from S3, counts word occurrences, and saves results as JSON to S3.
- **Reducer (1 task):** Reads all 3 mapper outputs from S3, merges the word counts, and saves the final result to S3.

All three programs are written in Go, containerized with Docker, and deployed via ECR/ECS/Fargate.

### Execution Flow

```
1. Splitter: hamlet.txt → chunk_0.txt, chunk_1.txt, chunk_2.txt
2. Mapper 0: chunk_0.txt → map_0.json (2181 unique words)
   Mapper 1: chunk_1.txt → map_1.json (2331 unique words)
   Mapper 2: chunk_2.txt → map_2.json (2238 unique words)
3. Reducer: map_0.json + map_1.json + map_2.json → final_counts.json (4702 total tokens)
```

### Performance Results

| Step | Time (seconds) |
|------|---------------|
| Single Machine (Python) | 0.092 |
| Splitter | 0.247 |
| 3 Mappers (parallel) | 0.184 |
| Reducer | 0.193 |
| **MapReduce Total** | **0.624** |

### Key Observations

1. **Single machine is ~6.8x faster** for this small file (0.092s vs 0.624s). This is expected because the overhead of network communication, S3 read/write operations, ECS task startup latency, and container scheduling overhead dominates the actual computation time for a 159KB file.

2. **Parallel mapping is effective.** The 3 mappers ran concurrently in just 0.184s, demonstrating that the workload was successfully distributed across multiple machines.

3. **Token count discrepancy (4701 vs 4702).** The single machine found 4701 unique words while MapReduce found 4702. Because the splitter divides by byte size rather than word boundaries, a word may be split across two chunks and treated as two separate tokens.

4. **MapReduce shines at scale.** While slower for small files, MapReduce becomes essential when files grow to petabyte scale where a single machine cannot process the data in reasonable time. Adding more mappers scales horizontally.

### Thinking Questions

- **What if a mapper fails?** We could implement retry logic or health checks. A coordinator could detect failure and reassign the chunk to a new mapper.
- **How to scale to 100 mappers?** The splitter would need to divide the file into 100 chunks. ECS can launch 100 mapper tasks. A coordinator service would be helpful to manage the workflow automatically.
- **Challenging part of manual coordination?** Manually obtaining 5 public IPs and calling each service in the correct order is tedious and error-prone. An orchestration tool (like AWS Step Functions) would automate this pipeline.