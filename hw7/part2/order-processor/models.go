package main

import "time"

type Item struct {
	ItemID   string  `json:"item_id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"`
	Items      []Item    `json:"items"`
	CreatedAt  time.Time `json:"created_at"`
}

// SNS wraps the order JSON in its own envelope when publishing to SQS.
// The actual order data is inside the "Message" field as a JSON string.
type SNSMessage struct {
	Message string `json:"Message"`
}