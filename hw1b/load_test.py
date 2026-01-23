import requests
import time
import matplotlib.pyplot as plt
import numpy as np

def load_test(url, duration_seconds=30):
    """
    Send continuous GET requests to a URL for a specified duration
    and collect response time data.

    Args:
        url: The endpoint to test
        duration_seconds: How long to run the test (default 30s)

    Returns:
        List of response times in milliseconds
    """
    response_times = []  # Store all response times
    start_time = time.time()  # Record when test started
    end_time = start_time + duration_seconds  # Calculate when to stop

    print(f"Starting load test for {duration_seconds} seconds...")

    # Keep sending requests until time is up
    while time.time() < end_time:
        try:
            # Measure how long each request takes
            start_request = time.time()
            response = requests.get(url, timeout=10)
            end_request = time.time()

            # Calculate response time in milliseconds
            response_time = (end_request - start_request) * 1000
            response_times.append(response_time)

            # Print progress
            if response.status_code == 200:
                print(f"Request {len(response_times)}: {response_time:.2f}ms")
            else:
                print(f"Request {len(response_times)}: Failed with status {response.status_code}")

        except requests.exceptions.RequestException as e:
            print(f"Request failed: {e}")

    return response_times

# Target URL - EC2 server endpoint
EC2_URL = "http://44.252.106.108:8080/albums"

# Run the load test
response_times = load_test(EC2_URL)

# Create a figure with 2 subplots (2 rows, 1 column)
plt.figure(figsize=(12, 8))

# Shows distribution of response times (how many requests fell into each time range)
plt.subplot(2, 1, 1)
plt.hist(response_times, bins=50, alpha=0.7, color='blue')
plt.xlabel('Response Time (ms)')
plt.ylabel('Frequency')
plt.title('Distribution of Response Times')

# Shows response time for each request over time (to spot patterns/spikes)
plt.subplot(2, 1, 2)
plt.scatter(range(len(response_times)), response_times, alpha=0.6)
plt.xlabel('Request Number')
plt.ylabel('Response Time (ms)')
plt.title('Response Times Over Time')

# Adjust layout and save
plt.tight_layout()
plt.savefig('load_test_results.png')
plt.show()

# Print statistics
print(f"\nStatistics:")
print(f"Total requests: {len(response_times)}")
print(f"Average response time: {np.mean(response_times):.2f}ms")
print(f"Median response time: {np.median(response_times):.2f}ms")
print(f"95th percentile: {np.percentile(response_times, 95):.2f}ms")
print(f"99th percentile: {np.percentile(response_times, 99):.2f}ms")
print(f"Max response time: {max(response_times):.2f}ms")