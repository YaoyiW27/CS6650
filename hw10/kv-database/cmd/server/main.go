package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"kv-database/internal/store"
)

var kvStore *store.KVStore

func main() {
	kvStore = store.NewKVStore()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/set", handleSet)
	http.HandleFunc("/get", handleGet)
	http.HandleFunc("/local_read", handleLocalRead)
	http.HandleFunc("/health", handleHealth)

	log.Printf("KV Store listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleSet — PUT /set?key=xxx&value=yyy
func handleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")

	if key == "" {
		http.Error(w, "key cannot be empty", http.StatusBadRequest)
		return
	}

	version := kvStore.Set(key, value)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":     key,
		"version": version,
	})
}

// handleGet — GET /get?key=xxx
func handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key cannot be empty", http.StatusBadRequest)
		return
	}

	entry, exists := kvStore.Get(key)
	if !exists {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"value":   entry.Value,
		"version": entry.Version,
	})
}

// handleLocalRead — GET /local_read?key=xxx
// "Sneaky" endpoint: always reads local store, no forwarding.
// Used in tests to observe inconsistency windows.
func handleLocalRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key cannot be empty", http.StatusBadRequest)
		return
	}

	entry, exists := kvStore.Get(key)
	if !exists {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"value":   entry.Value,
		"version": entry.Version,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}