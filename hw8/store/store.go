package store

import (
	"context"
	"time"
)

// CartItem represents a single item in a shopping cart.
type CartItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// Cart represents a shopping cart with its items.
type Cart struct {
	ID         int        `json:"shopping_cart_id"`
	CustomerID int        `json:"customer_id"`
	CreatedAt  time.Time  `json:"created_at"`
	Items      []CartItem `json:"items"`
}

// CartStore defines the interface for shopping cart persistence.
// Both MySQL and DynamoDB implementations satisfy this interface.
type CartStore interface {
	// CreateCart creates a new shopping cart and returns its ID.
	CreateCart(ctx context.Context, customerID int) (int, error)

	// GetCart retrieves a cart by ID, including all its items.
	GetCart(ctx context.Context, cartID int) (*Cart, error)

	// AddItem adds or updates an item in an existing cart.
	// If the product already exists in the cart, quantity is added.
	AddItem(ctx context.Context, cartID int, productID int, quantity int) error

	// Close cleans up any resources (e.g., DB connection pool).
	Close() error
}