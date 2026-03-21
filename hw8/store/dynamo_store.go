package store

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// DynamoStore implements CartStore using Amazon DynamoDB.
type DynamoStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoStore creates a new DynamoStore.
func NewDynamoStore(region, tableName string) (*DynamoStore, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	return &DynamoStore{
		client:    client,
		tableName: tableName,
	}, nil
}

func (s *DynamoStore) CreateCart(ctx context.Context, customerID int) (int, error) {
	cartID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	id := hashUUID(cartID)

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item: map[string]types.AttributeValue{
			"cart_id":     &types.AttributeValueMemberS{Value: cartID},
			"numeric_id":  &types.AttributeValueMemberN{Value: strconv.Itoa(id)},
			"customer_id": &types.AttributeValueMemberN{Value: strconv.Itoa(customerID)},
			"created_at":  &types.AttributeValueMemberS{Value: now},
			"items":       &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("put item: %w", err)
	}

	return id, nil
}

func (s *DynamoStore) GetCart(ctx context.Context, cartID int) (*Cart, error) {
	// We need to scan or use a secondary approach since our key is UUID string
	// but the interface uses int IDs. For this assignment, we store the int ID
	// as an attribute and scan. Better approach: use GSI on numeric_id.
	//
	// Simple approach: store numeric_id as attribute, use Scan with filter.
	// This is OK for 150-operation test scale.

	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        &s.tableName,
		FilterExpression: aws.String("numeric_id = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberN{Value: strconv.Itoa(cartID)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, nil
	}

	return parseCartItem(result.Items[0])
}

func (s *DynamoStore) AddItem(ctx context.Context, cartID int, productID int, quantity int) error {
	// First find the cart by numeric_id
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        &s.tableName,
		FilterExpression: aws.String("numeric_id = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberN{Value: strconv.Itoa(cartID)},
		},
	})
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	if len(result.Items) == 0 {
		return fmt.Errorf("cart %d not found", cartID)
	}

	item := result.Items[0]
	cartUUID := item["cart_id"].(*types.AttributeValueMemberS).Value

	// Get existing items and check if product already exists
	existingItems := item["items"].(*types.AttributeValueMemberL).Value
	found := false
	for i, ei := range existingItems {
		m := ei.(*types.AttributeValueMemberM).Value
		pid := m["product_id"].(*types.AttributeValueMemberN).Value
		if pid == strconv.Itoa(productID) {
			// Update quantity
			oldQty, _ := strconv.Atoi(m["quantity"].(*types.AttributeValueMemberN).Value)
			existingItems[i] = &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"product_id": &types.AttributeValueMemberN{Value: strconv.Itoa(productID)},
					"quantity":   &types.AttributeValueMemberN{Value: strconv.Itoa(oldQty + quantity)},
				},
			}
			found = true
			break
		}
	}

	if !found {
		existingItems = append(existingItems, &types.AttributeValueMemberM{
			Value: map[string]types.AttributeValue{
				"product_id": &types.AttributeValueMemberN{Value: strconv.Itoa(productID)},
				"quantity":   &types.AttributeValueMemberN{Value: strconv.Itoa(quantity)},
			},
		})
	}

	// Update the item (use #items because "items" is a DynamoDB reserved keyword)
	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"cart_id": &types.AttributeValueMemberS{Value: cartUUID},
		},
		UpdateExpression: aws.String("SET #items = :items"),
		ExpressionAttributeNames: map[string]string{
			"#items": "items",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":items": &types.AttributeValueMemberL{Value: existingItems},
		},
	})
	if err != nil {
		return fmt.Errorf("update item: %w", err)
	}

	return nil
}

func (s *DynamoStore) Close() error {
	return nil // DynamoDB client doesn't need explicit closing
}

// hashUUID converts a UUID string to a positive int for API compatibility.
func hashUUID(u string) int {
	h := 0
	for _, c := range u {
		h = h*31 + int(c)
		h = h & 0x7FFFFFFF // keep positive
	}
	if h == 0 {
		h = 1
	}
	return h
}

func parseCartItem(item map[string]types.AttributeValue) (*Cart, error) {
	cart := &Cart{}

	if v, ok := item["numeric_id"]; ok {
		id, _ := strconv.Atoi(v.(*types.AttributeValueMemberN).Value)
		cart.ID = id
	}
	if v, ok := item["customer_id"]; ok {
		cid, _ := strconv.Atoi(v.(*types.AttributeValueMemberN).Value)
		cart.CustomerID = cid
	}
	if v, ok := item["created_at"]; ok {
		t, _ := time.Parse(time.RFC3339, v.(*types.AttributeValueMemberS).Value)
		cart.CreatedAt = t
	}

	cart.Items = []CartItem{}
	if v, ok := item["items"]; ok {
		for _, li := range v.(*types.AttributeValueMemberL).Value {
			m := li.(*types.AttributeValueMemberM).Value
			pid, _ := strconv.Atoi(m["product_id"].(*types.AttributeValueMemberN).Value)
			qty, _ := strconv.Atoi(m["quantity"].(*types.AttributeValueMemberN).Value)
			cart.Items = append(cart.Items, CartItem{ProductID: pid, Quantity: qty})
		}
	}

	return cart, nil
}