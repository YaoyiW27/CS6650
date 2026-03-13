package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func startWorkers(sqsClient *sqs.SQS, queueURL string, numWorkers int) {
	// Buffered channel acts as a job queue for worker goroutines
	jobs := make(chan *sqs.Message, numWorkers*2)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for msg := range jobs {
				processMessage(sqsClient, queueURL, msg, id)
			}
		}(i)
	}

	log.Printf("Started %d worker goroutines, polling SQS...", numWorkers)

	// Poll SQS forever
	for {
		result, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(20), // Long polling
		})
		if err != nil {
			log.Printf("Error receiving messages: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, msg := range result.Messages {
			jobs <- msg
		}
	}
}

func processMessage(sqsClient *sqs.SQS, queueURL string, msg *sqs.Message, workerID int) {
	// Parse SNS envelope
	var snsMsg SNSMessage
	if err := json.Unmarshal([]byte(*msg.Body), &snsMsg); err != nil {
		log.Printf("[Worker %d] Failed to parse SNS message: %v", workerID, err)
		deleteMessage(sqsClient, queueURL, msg)
		return
	}

	// Parse order from SNS Message field
	var order Order
	if err := json.Unmarshal([]byte(snsMsg.Message), &order); err != nil {
		log.Printf("[Worker %d] Failed to parse order: %v", workerID, err)
		deleteMessage(sqsClient, queueURL, msg)
		return
	}

	// Simulate 3-second payment processing
	log.Printf("[Worker %d] Processing order %s for customer %d", workerID, order.OrderID, order.CustomerID)
	time.Sleep(3 * time.Second)
	log.Printf("[Worker %d] Order %s completed", workerID, order.OrderID)

	// Delete message from queue after successful processing
	deleteMessage(sqsClient, queueURL, msg)
}

func deleteMessage(sqsClient *sqs.SQS, queueURL string, msg *sqs.Message) {
	_, err := sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("Failed to delete message: %v", err)
	}
}