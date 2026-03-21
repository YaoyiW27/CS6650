package main

import (
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"hw8/handler"
	"hw8/store"
)

func main() {
	// Read config from environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnvInt("DB_PORT", 3306)
	dbUser := getEnv("DB_USER", "admin")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "shopping")
	port := getEnv("PORT", "8080")
	storeType := getEnv("STORE_TYPE", "mysql") // "mysql" or "dynamodb"
	dynamoTable := getEnv("DYNAMO_TABLE", "")
	awsRegion := getEnv("AWS_REGION", "us-west-2")

	var s store.CartStore
	var err error

	switch storeType {
	case "mysql":
		log.Printf("Connecting to MySQL at %s:%d/%s", dbHost, dbPort, dbName)
		s, err = store.NewMySQLStore(dbHost, dbPort, dbUser, dbPass, dbName)
		if err != nil {
			log.Fatalf("Failed to init MySQL store: %v", err)
		}
	case "dynamodb":
		log.Printf("Connecting to DynamoDB table %s in %s", dynamoTable, awsRegion)
		s, err = store.NewDynamoStore(awsRegion, dynamoTable)
		if err != nil {
			log.Fatalf("Failed to init DynamoDB store: %v", err)
		}
	default:
		log.Fatalf("Unknown STORE_TYPE: %s", storeType)
	}
	defer s.Close()

	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "store": storeType})
	})

	h := handler.NewCartHandler(s)
	h.RegisterRoutes(r)

	log.Printf("Server starting on :%s with %s store", port, storeType)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}