package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

func main() {
	// Get SNS topic ARN from environment variable
	topicARN := os.Getenv("SNS_TOPIC_ARN")
	if topicARN == "" {
		log.Fatal("SNS_TOPIC_ARN environment variable is required")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-west-2"
	}

	// Initialize AWS session and SNS client
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}
	snsClient := sns.New(sess)

	// Register routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/orders/sync", syncOrderHandler)
	http.HandleFunc("/orders/async", asyncOrderHandler(snsClient, topicARN))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Order receiver starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}