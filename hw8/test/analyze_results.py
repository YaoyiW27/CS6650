"""
Analyze combined_results.json and print precise metrics for the report.
Usage: python analyze_results.py
"""

import json
import statistics

with open("combined_results.json") as f:
    data = json.load(f)

for db in ["mysql", "dynamodb"]:
    print(f"\n{'='*50}")
    print(f"  {db.upper()}")
    print(f"{'='*50}")

    db_results = [r for r in data if r["database"] == db]
    all_times = [r["response_time"] for r in db_results]
    all_times.sort()

    n = len(all_times)
    p50 = all_times[int(n * 0.50)]
    p95 = all_times[int(n * 0.95)]
    p99 = all_times[int(n * 0.99)]
    avg = statistics.mean(all_times)
    successes = sum(1 for r in db_results if r["success"])

    print(f"  Total ops:    {n}")
    print(f"  Success rate: {successes}/{n} ({successes/n*100:.1f}%)")
    print(f"  Avg:  {avg:.1f}ms")
    print(f"  P50:  {p50:.1f}ms")
    print(f"  P95:  {p95:.1f}ms")
    print(f"  P99:  {p99:.1f}ms")
    print(f"  Min:  {min(all_times):.1f}ms")
    print(f"  Max:  {max(all_times):.1f}ms")

    for op in ["create_cart", "add_items", "get_cart"]:
        times = sorted([r["response_time"] for r in db_results if r["operation"] == op])
        s = sum(1 for r in db_results if r["operation"] == op and r["success"])
        print(f"\n  {op}:")
        print(f"    Count: {len(times)}, Success: {s}/{len(times)}")
        print(f"    Avg: {statistics.mean(times):.1f}ms, Min: {min(times):.1f}ms, Max: {max(times):.1f}ms")