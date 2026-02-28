"""
Locust load test for Resilient Microservices Demo.

Usage:
  # Test a specific phase:
  locust -f locustfile.py --host=http://localhost:8080 --tags phase1

  # Run all phases sequentially (headless):
  ./scripts/run_tests.sh

  # Web UI mode (pick phase from dropdown):
  locust -f locustfile.py --host=http://localhost:8080
"""

from locust import HttpUser, task, tag, between, events
import time
import json


class OrderUser(HttpUser):
    wait_time = between(0.1, 0.5)  # short wait to generate high load

    # Phase 1: No Resilience
    @tag("phase1", "all")
    @task
    def phase1_create_order(self):
        self.client.post("/orders/phase1", name="Phase1 - No Resilience")

    # Phase 2: Fail Fast
    @tag("phase2", "all")
    @task
    def phase2_create_order(self):
        self.client.post("/orders/phase2", name="Phase2 - Fail Fast")

    # Phase 3: Circuit Breaker
    @tag("phase3", "all")
    @task
    def phase3_create_order(self):
        self.client.post("/orders/phase3", name="Phase3 - Circuit Breaker")

    # Phase 4: Bulkhead + All
    @tag("phase4", "all")
    @task
    def phase4_create_order(self):
        self.client.post("/orders/phase4", name="Phase4 - Bulkhead")

    # Health Check (lightweight, always available)
    @tag("health")
    @task
    def check_health(self):
        self.client.get("/health", name="Health Check")