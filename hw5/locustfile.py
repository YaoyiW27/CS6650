import random
from locust import HttpUser, FastHttpUser, task, between

# class ProductUser(HttpUser):
#     wait_time = between(0.1, 0.5)

#     # counter for creating unique products
#     product_counter = 0

#     def on_start(self):
#         """Create a product on start so GET has something to fetch"""
#         ProductUser.product_counter += 1
#         self.my_product_id = ProductUser.product_counter
#         self.client.post(
#             f"/products/{self.my_product_id}/details",
#             json={
#                 "product_id": self.my_product_id,
#                 "sku": f"SKU-{self.my_product_id}",
#                 "manufacturer": "TestCorp",
#                 "category_id": 1,
#                 "weight": 100,
#                 "some_other_id": 1,
#             },
#         )

#     @task(3)
#     def get_product(self):
#         """GET is more common in real e-commerce (browsing > creating)"""
#         self.client.get(f"/products/{self.my_product_id}")

#     @task(1)
#     def create_product(self):
#         """POST to create/update a product"""
#         ProductUser.product_counter += 1
#         pid = ProductUser.product_counter
#         self.client.post(
#             f"/products/{pid}/details",
#             json={
#                 "product_id": pid,
#                 "sku": f"SKU-{pid}",
#                 "manufacturer": "TestCorp",
#                 "category_id": random.randint(1, 10),
#                 "weight": random.randint(0, 5000),
#                 "some_other_id": random.randint(1, 100),
#             },
#         )


class ProductFastUser(FastHttpUser):
    """Same tests but using FastHttpUser for comparison"""
    wait_time = between(0.1, 0.5)

    product_counter = 0

    def on_start(self):
        ProductFastUser.product_counter += 1
        self.my_product_id = ProductFastUser.product_counter
        self.client.post(
            f"/products/{self.my_product_id}/details",
            json={
                "product_id": self.my_product_id,
                "sku": f"SKU-{self.my_product_id}",
                "manufacturer": "TestCorp",
                "category_id": 1,
                "weight": 100,
                "some_other_id": 1,
            },
        )

    @task(3)
    def get_product(self):
        self.client.get(f"/products/{self.my_product_id}")

    @task(1)
    def create_product(self):
        ProductFastUser.product_counter += 1
        pid = ProductFastUser.product_counter
        self.client.post(
            f"/products/{pid}/details",
            json={
                "product_id": pid,
                "sku": f"SKU-{pid}",
                "manufacturer": "TestCorp",
                "category_id": random.randint(1, 10),
                "weight": random.randint(0, 5000),
                "some_other_id": random.randint(1, 100),
            },
        )