package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		log.Fatalf("unable to load AWS config: %v", err)
	}
	s3Client = s3.NewFromConfig(cfg)
}

func splitHandler(w http.ResponseWriter, r *http.Request) {
	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")
	chunks := 3

	if bucket == "" || key == "" {
		http.Error(w, "missing bucket or key param", http.StatusBadRequest)
		return
	}

	// Read file from S3
	result, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get S3 object: %v", err), http.StatusInternalServerError)
		return
	}
	defer result.Body.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read body: %v", err), http.StatusInternalServerError)
		return
	}

	content := string(body)
	chunkSize := len(content) / chunks
	var chunkKeys []string

	for i := 0; i < chunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == chunks-1 {
			end = len(content) // last chunk gets the remainder
		}

		chunkKey := fmt.Sprintf("chunks/chunk_%d.txt", i)
		_, err := s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(chunkKey),
			Body:   strings.NewReader(content[start:end]),
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to upload chunk %d: %v", i, err), http.StatusInternalServerError)
			return
		}
		chunkKeys = append(chunkKeys, chunkKey)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bucket": bucket,
		"chunks": chunkKeys,
	})
}

func main() {
	http.HandleFunc("/split", splitHandler)
	log.Println("Splitter listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}