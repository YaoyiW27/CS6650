package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode"

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

func mapHandler(w http.ResponseWriter, r *http.Request) {
	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")
	outputKey := r.URL.Query().Get("output")

	if bucket == "" || key == "" || outputKey == "" {
		http.Error(w, "missing bucket, key, or output param", http.StatusBadRequest)
		return
	}

	// Read chunk from S3
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

	// Count words
	counts := make(map[string]int)
	words := strings.FieldsFunc(string(body), func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	})
	for _, word := range words {
		counts[strings.ToLower(word)]++
	}

	// Save result as JSON to S3
	jsonData, _ := json.Marshal(counts)
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(outputKey),
		Body:   strings.NewReader(string(jsonData)),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to upload result: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bucket": bucket,
		"output": outputKey,
		"words":  len(counts),
	})
}

func main() {
	http.HandleFunc("/map", mapHandler)
	log.Println("Mapper listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}