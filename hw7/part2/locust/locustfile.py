from locust import HttpUser, task, between
import random
import json


class OrderUser(HttpUser):
    wait_time = between(0.1, 0.5)  # 100-500ms between requests

    def make_order(self):
        return {
            "customer_id": random.randint(1, 1000),
            "items": [
                {
                    "item_id": f"ITEM-{random.randint(1, 100)}",
                    "name": f"Product {random.randint(1, 50)}",
                    "price": round(random.uniform(5.0, 100.0), 2),
                    "quantity": random.randint(1, 5),
                }
            ],
        }

    @task
    def sync_order(self):
        self.client.post(
            "/orders/sync",
            json=self.make_order(),
            headers={"Content-Type": "application/json"},
        )


class AsyncOrderUser(HttpUser):
    wait_time = between(0.1, 0.5)

    def make_order(self):
        return {
            "customer_id": random.randint(1, 1000),
            "items": [
                {
                    "item_id": f"ITEM-{random.randint(1, 100)}",
                    "name": f"Product {random.randint(1, 50)}",
                    "price": round(random.uniform(5.0, 100.0), 2),
                    "quantity": random.randint(1, 5),
                }
            ],
        }

    @task
    def async_order(self):
        self.client.post(
            "/orders/async",
            json=self.make_order(),
            headers={"Content-Type": "application/json"},
        )