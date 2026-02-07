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

func reduceHandler(w http.ResponseWriter, r *http.Request) {
	bucket := r.URL.Query().Get("bucket")
	keys := r.URL.Query().Get("keys") // comma-separated keys

	if bucket == "" || keys == "" {
		http.Error(w, "missing bucket or keys param", http.StatusBadRequest)
		return
	}

	keyList := strings.Split(keys, ",")
	finalCounts := make(map[string]int)

	// Read and merge all mapper outputs
	for _, key := range keyList {
		result, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(strings.TrimSpace(key)),
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get %s: %v", key, err), http.StatusInternalServerError)
			return
		}

		body, err := io.ReadAll(result.Body)
		result.Body.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read %s: %v", key, err), http.StatusInternalServerError)
			return
		}

		var counts map[string]int
		if err := json.Unmarshal(body, &counts); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse %s: %v", key, err), http.StatusInternalServerError)
			return
		}

		for word, count := range counts {
			finalCounts[word] += count
		}
	}

	// Save final result to S3
	outputKey := "results/final_counts.json"
	jsonData, _ := json.Marshal(finalCounts)
	_, err := s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
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
		"bucket":      bucket,
		"output":      outputKey,
		"total_words": len(finalCounts),
	})
}

func main() {
	http.HandleFunc("/reduce", reduceHandler)
	log.Println("Reducer listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}