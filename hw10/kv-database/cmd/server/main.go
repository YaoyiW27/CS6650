package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"kv-database/internal/replication"
	"kv-database/internal/store"
)

var (
	kvStore *store.KVStore
	replMgr *replication.ReplicationManager
)

func main() {
	kvStore = store.NewKVStore()

	// --- Parse environment variables ---
	role := os.Getenv("ROLE") // "leader" or "follower"
	if role == "" {
		role = "standalone"
	}

	// PEERS is a comma-separated list: "http://follower1:8080,http://follower2:8080,..."
	peersStr := os.Getenv("PEERS")
	var peers []string
	if peersStr != "" {
		peers = strings.Split(peersStr, ",")
	}

	w := getEnvInt("W", 1)
	r := getEnvInt("R", 1)
	n := getEnvInt("N", 5)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	replMgr = &replication.ReplicationManager{
		Store: kvStore,
		Config: replication.Config{
			Role:  role,
			Peers: peers,
			W:     w,
			R:     r,
			N:     n,
		},
	}

	// --- Client-facing endpoints ---
	http.HandleFunc("/set", handleSet)
	http.HandleFunc("/get", handleGet)
	http.HandleFunc("/local_read", handleLocalRead)
	http.HandleFunc("/health", handleHealth)

	// --- Internal endpoints (node-to-node) ---
	http.HandleFunc("/internal/set", replMgr.HandleInternalSet)
	http.HandleFunc("/internal/get", replMgr.HandleInternalGet)

	log.Printf("KV Store [%s] listening on :%s (W=%d, R=%d, N=%d, peers=%v)",
		role, port, w, r, n, peers)
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

	switch replMgr.Config.Role {
	case "leader":
		// Leader handles the write + replication
		version, err := replMgr.LeaderSet(key, value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key":     key,
			"version": version,
		})

	case "follower":
		// Followers should NOT receive client writes in leader-follower mode
		http.Error(w, "writes must go to the leader", http.StatusForbidden)

	default:
		// Standalone mode (no replication)
		version := kvStore.Set(key, value)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key":     key,
			"version": version,
		})
	}
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

	switch replMgr.Config.Role {
	case "leader", "follower":
		// Use quorum read (reads from R nodes, returns highest version)
		entry, found := replMgr.QuorumRead(key)
		if !found {
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"value":   entry.Value,
			"version": entry.Version,
		})

	default:
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
}

// handleLocalRead — GET /local_read?key=xxx
// Always reads from THIS node only. No quorum. Used for testing inconsistency.
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

func getEnvInt(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}