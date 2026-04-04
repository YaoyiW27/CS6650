#!/bin/bash
# Run all 16 load test combinations (4 configs × 4 write ratios)
# Results go into results/ directory as CSV files.

set -e
mkdir -p results

TOTAL=2000
CONCURRENCY=20
KEYS=10
WRITE_PCTS="1 10 50 90"

WRITE_ADDR="http://localhost:8080"
READ_ADDRS="http://localhost:8080,http://localhost:8081,http://localhost:8082,http://localhost:8083,http://localhost:8084"

run_tests() {
    local db_type=$1
    echo ""
    echo "=========================================="
    echo "  Testing: $db_type"
    echo "=========================================="

    for wpct in $WRITE_PCTS; do
        echo ""
        echo "--- $db_type / write=$wpct% read=$((100-wpct))% ---"
        go run cmd/loadtest/main.go \
            -db-type "$db_type" \
            -write-addr "$WRITE_ADDR" \
            -read-addrs "$READ_ADDRS" \
            -write-pct "$wpct" \
            -total "$TOTAL" \
            -concurrency "$CONCURRENCY" \
            -keys "$KEYS" \
            -output "results/${db_type}_w${wpct}.csv"
        echo "Done."
        sleep 2
    done
}

echo "================================================"
echo "  KV Database Load Test Suite"
echo "  Total requests per run: $TOTAL"
echo "  Concurrency: $CONCURRENCY"
echo "  Keys: $KEYS"
echo "================================================"

# ---- Config 1: Leader-Follower W=5, R=1 ----
echo ""
echo ">>> Starting Leader-Follower W=5, R=1 ..."
docker compose -f docker-compose-leaderless.yml down 2>/dev/null || true
docker compose -f docker-compose-leader.yml down 2>/dev/null || true

# Update W and R in compose (we'll use env override)
W=5 R=1 docker compose -f docker-compose-leader.yml up --build -d
sleep 5
run_tests "leader-w5r1"

# ---- Config 2: Leader-Follower W=1, R=5 ----
echo ""
echo ">>> Starting Leader-Follower W=1, R=5 ..."
docker compose -f docker-compose-leader.yml down

# We need to change W/R — simplest: use a separate compose or override env
# For now, let's create temp overrides
cat > /tmp/docker-compose-override.yml <<EOF
services:
  leader:
    environment:
      - ROLE=leader
      - PEERS=http://follower1:8080,http://follower2:8080,http://follower3:8080,http://follower4:8080
      - W=1
      - R=5
      - N=5
  follower1:
    environment:
      - ROLE=follower
      - PEERS=http://leader:8080,http://follower2:8080,http://follower3:8080,http://follower4:8080
      - W=1
      - R=5
      - N=5
  follower2:
    environment:
      - ROLE=follower
      - PEERS=http://leader:8080,http://follower1:8080,http://follower3:8080,http://follower4:8080
      - W=1
      - R=5
      - N=5
  follower3:
    environment:
      - ROLE=follower
      - PEERS=http://leader:8080,http://follower1:8080,http://follower2:8080,http://follower4:8080
      - W=1
      - R=5
      - N=5
  follower4:
    environment:
      - ROLE=follower
      - PEERS=http://leader:8080,http://follower1:8080,http://follower2:8080,http://follower3:8080
      - W=1
      - R=5
      - N=5
EOF

docker compose -f docker-compose-leader.yml -f /tmp/docker-compose-override.yml up --build -d
sleep 5
run_tests "leader-w1r5"

# ---- Config 3: Leader-Follower W=3, R=3 ----
echo ""
echo ">>> Starting Leader-Follower W=3, R=3 ..."
docker compose -f docker-compose-leader.yml -f /tmp/docker-compose-override.yml down

cat > /tmp/docker-compose-override.yml <<EOF
services:
  leader:
    environment:
      - ROLE=leader
      - PEERS=http://follower1:8080,http://follower2:8080,http://follower3:8080,http://follower4:8080
      - W=3
      - R=3
      - N=5
  follower1:
    environment:
      - ROLE=follower
      - PEERS=http://leader:8080,http://follower2:8080,http://follower3:8080,http://follower4:8080
      - W=3
      - R=3
      - N=5
  follower2:
    environment:
      - ROLE=follower
      - PEERS=http://leader:8080,http://follower1:8080,http://follower3:8080,http://follower4:8080
      - W=3
      - R=3
      - N=5
  follower3:
    environment:
      - ROLE=follower
      - PEERS=http://leader:8080,http://follower1:8080,http://follower2:8080,http://follower4:8080
      - W=3
      - R=3
      - N=5
  follower4:
    environment:
      - ROLE=follower
      - PEERS=http://leader:8080,http://follower1:8080,http://follower2:8080,http://follower3:8080
      - W=3
      - R=3
      - N=5
EOF

docker compose -f docker-compose-leader.yml -f /tmp/docker-compose-override.yml up --build -d
sleep 5
run_tests "leader-w3r3"

# ---- Config 4: Leaderless W=5, R=1 ----
echo ""
echo ">>> Starting Leaderless W=5, R=1 ..."
docker compose -f docker-compose-leader.yml -f /tmp/docker-compose-override.yml down
docker compose -f docker-compose-leaderless.yml up --build -d
sleep 5
run_tests "leaderless"

# Cleanup
docker compose -f docker-compose-leaderless.yml down
rm -f /tmp/docker-compose-override.yml

echo ""
echo "================================================"
echo "  All tests complete! Results in results/"
echo "================================================"
ls -la results/