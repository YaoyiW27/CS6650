# CS6650 HW5 - Product API

## Product API

Implements two endpoints from the OpenAPI specification (`api.yaml`), with in-memory storage using a hashmap protected by a read-write mutex for thread safety.

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