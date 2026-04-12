package config

import (
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	AWSRegion   string
	S3Bucket    string
	SQSQueueURL string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/albumstore?sslmode=disable"),
		AWSRegion:   getEnv("AWS_REGION", "us-west-2"),
		S3Bucket:    getEnv("S3_BUCKET", ""),
		SQSQueueURL: getEnv("SQS_QUEUE_URL", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}