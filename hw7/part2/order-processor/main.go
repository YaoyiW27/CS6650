package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func main() {
	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		log.Fatal("SQS_QUEUE_URL environment variable is required")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-west-2"
	}

	// Number of worker goroutines — adjust for Phase 5 scaling tests
	numWorkers := 1
	if val := os.Getenv("NUM_WORKERS"); val != "" {
		n, err := strconv.Atoi(val)
		if err == nil && n > 0 {
			numWorkers = n
		}
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}
	sqsClient := sqs.New(sess)

	// Health check endpoint — ECS needs this even though we're not serving HTTP traffic
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	log.Printf("Order processor starting with %d workers, queue: %s", numWorkers, queueURL)
	startWorkers(sqsClient, queueURL, numWorkers)
}