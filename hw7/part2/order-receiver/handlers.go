package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/google/uuid"
)

// Buffered channel to simulate payment processor bottleneck
// Capacity 1 = only 1 order can be processed at a time, each takes 3 seconds

var paymentSlot = make(chan struct{}, 1)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func syncOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	order.OrderID = uuid.NewString()
	order.Status = "processing"
	order.CreatedAt = time.Now().UTC()

	if err := verifyPayment(order); err != nil {
		writeJSON(w, http.StatusPaymentRequired, map[string]string{"error": "payment verification failed"})
		return
	}

	order.Status = "completed"
	log.Printf("[SYNC] Order %s processed successfully for customer %d", order.OrderID, order.CustomerID)
	writeJSON(w, http.StatusOK, OrderResponse{
		OrderID: order.OrderID,
		Status:  order.Status,
		Message: "Order processed successfully",
	})
}

func asyncOrderHandler(snsClient *sns.SNS, topicARN string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var order Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		order.OrderID = uuid.NewString()
		order.Status = "pending"
		order.CreatedAt = time.Now().UTC()

		// Publish order to SNS topic
		orderJSON, err := json.Marshal(order)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to marshal order"})
			return
		}

		_, err = snsClient.Publish(&sns.PublishInput{
			TopicArn: aws.String(topicARN),
			Message:  aws.String(string(orderJSON)),
		    Subject:  aws.String(fmt.Sprintf("New Order: %s", order.OrderID)),
		})
		if err != nil {
			log.Printf("[ASYNC] Failed to publish order %s: %v", order.OrderID, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to publish order"})
			return
		}

		log.Printf("[ASYNC] Order %s accepted for customer %d", order.OrderID, order.CustomerID)

		writeJSON(w, http.StatusAccepted, OrderResponse{
			OrderID: order.OrderID,
			Status:  order.Status,
			Message: "Order accepted for async processing",
		})
	}
}

func verifyPayment(order Order) error {
	paymentSlot <- struct{}{} // Acquire payment slot
	defer func() { <-paymentSlot }() // Release payment slot after processing
	time.Sleep(3 * time.Second) // Simulate payment processing delay
	return nil // In real implementation, this would involve actual payment verification logic
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}