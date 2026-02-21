from locust import FastHttpUser, task, between
import random

SEARCH_TERMS = ["alpha", "beta", "gamma", "delta", "electronics", "books", "home", "sports", "toys", "food"]

class ProductSearchUser(FastHttpUser):
    wait_time = between(0.1, 0.5)

    @task
    def search_products(self):
        q = random.choice(SEARCH_TERMS)
        self.client.get(f"/products/search?q={q}")