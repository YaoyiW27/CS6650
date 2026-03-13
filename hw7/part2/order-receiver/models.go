package main

import "time"

type Item struct {
	ItemID  string  `json:"item_id"`
	Name    string  `json:"name"`
	Price   float64 `json:"price"`
	Quantity int     `json:"quantity"`
}

type Order struct {
	OrderID     string    `json:"order_id"`
	CustomerID  int    `json:"customer_id"`
	Status	    string    `json:"status"` // pending, processing, completed
	Items       []Item    `json:"items"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrderResponse struct {
	OrderID     string    `json:"order_id"`
	Status	    string    `json:"status"`
	Message     string    `json:"message"`
}
