package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLStore implements CartStore using Amazon RDS MySQL.
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore creates a new MySQLStore with connection pooling.
func NewMySQLStore(host string, port int, user, password, dbName string) (*MySQLStore, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=5s",
		user, password, host, port, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	// Auto-create tables
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &MySQLStore{db: db}, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS shopping_carts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			customer_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_customer_id (customer_id)
		) ENGINE=InnoDB`,

		`CREATE TABLE IF NOT EXISTS cart_items (
			id INT AUTO_INCREMENT PRIMARY KEY,
			cart_id INT NOT NULL,
			product_id INT NOT NULL,
			quantity INT NOT NULL DEFAULT 1,
			FOREIGN KEY (cart_id) REFERENCES shopping_carts(id) ON DELETE CASCADE,
			UNIQUE KEY uk_cart_product (cart_id, product_id)
		) ENGINE=InnoDB`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("create table failed: %w", err)
		}
	}
	return nil
}

func (s *MySQLStore) CreateCart(ctx context.Context, customerID int) (int, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO shopping_carts (customer_id) VALUES (?)", customerID)
	if err != nil {
		return 0, fmt.Errorf("insert cart: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}
	return int(id), nil
}

func (s *MySQLStore) GetCart(ctx context.Context, cartID int) (*Cart, error) {
	// Get cart header
	cart := &Cart{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, customer_id, created_at FROM shopping_carts WHERE id = ?",
		cartID).Scan(&cart.ID, &cart.CustomerID, &cart.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // not found
	}
	if err != nil {
		return nil, fmt.Errorf("query cart: %w", err)
	}

	// Get cart items
	rows, err := s.db.QueryContext(ctx,
		"SELECT product_id, quantity FROM cart_items WHERE cart_id = ?",
		cartID)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	cart.Items = []CartItem{}
	for rows.Next() {
		var item CartItem
		if err := rows.Scan(&item.ProductID, &item.Quantity); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		cart.Items = append(cart.Items, item)
	}

	return cart, rows.Err()
}

func (s *MySQLStore) AddItem(ctx context.Context, cartID int, productID int, quantity int) error {
	// Verify cart exists
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM shopping_carts WHERE id = ?)",
		cartID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check cart: %w", err)
	}
	if !exists {
		return fmt.Errorf("cart %d not found", cartID)
	}

	// Upsert: insert or update quantity if product already in cart
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity)`,
		cartID, productID, quantity)
	if err != nil {
		return fmt.Errorf("upsert item: %w", err)
	}

	return nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}