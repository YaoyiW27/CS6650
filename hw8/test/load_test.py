"""
HW8 Performance Test Script
Runs 150 operations: 50 create, 50 add items, 50 get cart
Usage: python load_test.py <base_url> <output_file>
Example: python load_test.py http://35.94.152.135:8080 mysql_test_results.json
"""

import requests
import time
import json
import sys
import random
from datetime import datetime, timezone


def run_test(base_url, output_file):
    results = []
    cart_ids = []

    print(f"Testing {base_url} → {output_file}")
    print("=" * 50)

    # Phase 1: Create 50 carts
    print("\n[Phase 1] Creating 50 carts...")
    for i in range(50):
        customer_id = random.randint(1, 10000)
        payload = {"customer_id": customer_id}

        start = time.time()
        try:
            resp = requests.post(f"{base_url}/shopping-carts", json=payload)
            elapsed = (time.time() - start) * 1000  # ms

            success = resp.status_code == 201
            if success:
                cart_ids.append(resp.json()["shopping_cart_id"])

            results.append({
                "operation": "create_cart",
                "response_time": round(elapsed, 2),
                "success": success,
                "status_code": resp.status_code,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            })
        except Exception as e:
            elapsed = (time.time() - start) * 1000
            results.append({
                "operation": "create_cart",
                "response_time": round(elapsed, 2),
                "success": False,
                "status_code": 0,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            })
            print(f"  ERROR on create #{i+1}: {e}")

        if (i + 1) % 10 == 0:
            print(f"  Created {i+1}/50")

    print(f"  Got {len(cart_ids)} cart IDs")

    # Phase 2: Add items to 50 carts
    print("\n[Phase 2] Adding items to 50 carts...")
    for i in range(50):
        cart_id = cart_ids[i % len(cart_ids)]
        payload = {
            "product_id": random.randint(1, 1000),
            "quantity": random.randint(1, 5),
        }

        start = time.time()
        try:
            resp = requests.post(
                f"{base_url}/shopping-carts/{cart_id}/items", json=payload
            )
            elapsed = (time.time() - start) * 1000

            results.append({
                "operation": "add_items",
                "response_time": round(elapsed, 2),
                "success": resp.status_code == 204,
                "status_code": resp.status_code,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            })
        except Exception as e:
            elapsed = (time.time() - start) * 1000
            results.append({
                "operation": "add_items",
                "response_time": round(elapsed, 2),
                "success": False,
                "status_code": 0,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            })
            print(f"  ERROR on add #{i+1}: {e}")

        if (i + 1) % 10 == 0:
            print(f"  Added {i+1}/50")

    # Phase 3: Get 50 carts
    print("\n[Phase 3] Getting 50 carts...")
    for i in range(50):
        cart_id = cart_ids[i % len(cart_ids)]

        start = time.time()
        try:
            resp = requests.get(f"{base_url}/shopping-carts/{cart_id}")
            elapsed = (time.time() - start) * 1000

            results.append({
                "operation": "get_cart",
                "response_time": round(elapsed, 2),
                "success": resp.status_code == 200,
                "status_code": resp.status_code,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            })
        except Exception as e:
            elapsed = (time.time() - start) * 1000
            results.append({
                "operation": "get_cart",
                "response_time": round(elapsed, 2),
                "success": False,
                "status_code": 0,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            })
            print(f"  ERROR on get #{i+1}: {e}")

        if (i + 1) % 10 == 0:
            print(f"  Got {i+1}/50")

    # Save results
    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)

    # Print summary
    print("\n" + "=" * 50)
    print(f"Total operations: {len(results)}")

    for op in ["create_cart", "add_items", "get_cart"]:
        op_results = [r for r in results if r["operation"] == op]
        times = [r["response_time"] for r in op_results]
        successes = sum(1 for r in op_results if r["success"])
        avg = sum(times) / len(times) if times else 0
        print(f"\n{op}:")
        print(f"  Count: {len(op_results)}, Success: {successes}/{len(op_results)}")
        print(f"  Avg: {avg:.1f}ms, Min: {min(times):.1f}ms, Max: {max(times):.1f}ms")

    print(f"\nResults saved to {output_file}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python load_test.py <base_url> <output_file>")
        print("Example: python load_test.py http://35.94.152.135:8080 mysql_test_results.json")
        sys.exit(1)

    run_test(sys.argv[1], sys.argv[2])