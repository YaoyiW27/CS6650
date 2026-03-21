"""
Combine MySQL and DynamoDB test results into a single file.
Usage: python combine_results.py
"""

import json

with open("mysql_test_results.json") as f:
    mysql = json.load(f)
with open("dynamodb_test_results.json") as f:
    dynamo = json.load(f)

# Tag each result with its database
for r in mysql:
    r["database"] = "mysql"
for r in dynamo:
    r["database"] = "dynamodb"

combined = mysql + dynamo

with open("combined_results.json", "w") as f:
    json.dump(combined, f, indent=2)

print(f"Combined {len(mysql)} MySQL + {len(dynamo)} DynamoDB = {len(combined)} total")
print("Saved to combined_results.json")