# CS6650 HW5 - Product API

## Product API

Implements two endpoints from the OpenAPI specification (`api.yaml`), with in-memory storage using a Go hashmap (`map[int]Product`) protected by a read-write mutex (`sync.RWMutex`) for thread safety under concurrent load.

### Endpoints

**GET /products/{productId}** — Retrieve a product by ID

**POST /products/{productId}/details** — Create or update a product's details. All fields are required and validated against the API spec constraints.

### Product Schema

| Field | Type | Constraint |
|-------|------|------------|
| product_id | int | >= 1, must match path param |
| sku | string | 1-100 characters |
| manufacturer | string | 1-200 characters |
| category_id | int | >= 1 |
| weight | int | >= 0 |
| some_other_id | int | >= 1 |

## How to Run

### Local

```bash
cd CS6650_2b_demo/src
go run main.go
# Server starts at http://localhost:8080
```

### Docker

```bash
cd CS6650_2b_demo/src
docker build -t product-api .
docker run -p 8080:8080 product-api
# Server starts at http://localhost:8080
```

### Deploy to AWS (Terraform)

1. Configure AWS credentials:
```bash
aws configure
aws configure set aws_session_token <YOUR_SESSION_TOKEN>
```

2. Deploy infrastructure:
```bash
cd CS6650_2b_demo/terraform
terraform init
terraform apply -auto-approve
```

3. Get public IP:
```bash
aws ec2 describe-network-interfaces \
  --network-interface-ids $(
    aws ecs describe-tasks \
      --cluster $(terraform output -raw ecs_cluster_name) \
      --tasks $(
        aws ecs list-tasks \
          --cluster $(terraform output -raw ecs_cluster_name) \
          --service-name $(terraform output -raw ecs_service_name) \
          --query 'taskArns[0]' --output text
      ) \
      --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" \
      --output text
  ) \
  --query 'NetworkInterfaces[0].Association.PublicIp' \
  --output text
```

4. Send requests to `http://<PUBLIC_IP>:8080`

5. Clean up:
```bash
terraform destroy -auto-approve
```

## API Examples (All Response Codes)

### POST /products/1/details → 204 No Content (Success)

```bash
curl -v -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":1,"sku":"ABC-123","manufacturer":"Acme","category_id":10,"weight":500,"some_other_id":99}'
```

### GET /products/1 → 200 OK

```bash
curl -v http://localhost:8080/products/1
```

Response:
```json
{"product_id":1,"sku":"ABC-123","manufacturer":"Acme","category_id":10,"weight":500,"some_other_id":99}
```

### GET /products/999 → 404 Not Found

```bash
curl -v http://localhost:8080/products/999
```

Response:
```json
{"error":"NOT_FOUND","message":"Product not found","details":"No product exists with the given productId"}
```

### GET /products/abc → 400 Bad Request (Invalid ID)

```bash
curl -v http://localhost:8080/products/abc
```

Response:
```json
{"error":"INVALID_INPUT","message":"Invalid product ID","details":"productId must be a positive integer (>= 1)"}
```

### POST with missing fields → 400 Bad Request

```bash
curl -v -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":1}'
```

Response:
```json
{"error":"INVALID_INPUT","message":"Invalid sku","details":"sku must be 1-100 characters and non-empty"}
```

### POST with mismatched product_id → 400 Bad Request

```bash
curl -v -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":2,"sku":"ABC","manufacturer":"Acme","category_id":10,"weight":500,"some_other_id":99}'
```

Response:
```json
{"error":"INVALID_INPUT","message":"product_id mismatch","details":"product_id in body must match path productId"}
```

## Load Testing Results (Locust)

Tests were run against the AWS-deployed server using both `HttpUser` and `FastHttpUser`.

GET requests were weighted 3x more than POST requests (`@task(3)` vs `@task(1)`) to simulate real e-commerce behavior where browsing is far more common than creating products.

### Results Summary

| Scenario | Users | Spawn Rate | HttpUser RPS | FastHttpUser RPS | Failures |
|----------|-------|------------|-------------|-----------------|----------|
| Normal | 50 | 5/s | ~156 | ~158 | 0% |
| Stress | 200 | 20/s | ~625 | ~633 | 0% |

### HttpUser vs FastHttpUser Analysis

There was virtually no difference between HttpUser and FastHttpUser in our tests. The reason is that the bottleneck is **network latency**, not client-side overhead. Each request travels from the local machine to AWS (~15-20ms round trip), which dominates the total response time. FastHttpUser uses a C-based HTTP library (geventhttpclient) that is more efficient than HttpUser's Python `requests` library, but when network latency accounts for 99% of the request time, client-side optimizations become negligible. The difference would only be visible if the server were running on localhost, where network latency is near zero and client overhead becomes the dominant factor.

### Tradeoffs: Data Structure Choice

A hashmap (`map[int]Product`) was chosen for in-memory storage because:

- **GET operations are O(1)** — In a real e-commerce system, reads (browsing products) vastly outnumber writes (adding products). Hashmap provides constant-time lookups, which is ideal for read-heavy workloads.
- **POST operations are O(1)** — Insertions and updates are also constant time.
- **Trade-off**: No ordering, no range queries, and data is lost on restart. A real system would use a database for persistence and potentially add caching (e.g., Redis) for frequently accessed products.

### Screenshots

See the `screenshots/` folder for detailed test evidence including:
- API response code examples (GET 200, 404, 400; POST 204, 400)
- Docker build and test
- Terraform deployment and CloudWatch logs
- Locust load test results (HttpUser normal/stress, FastHttpUser normal/stress)