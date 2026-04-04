package replication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"kv-database/internal/store"
)

// Config holds the replication parameters.
type Config struct {
	Role  string   // "leader" or "follower"
	Peers []string // addresses of all OTHER nodes, e.g. ["http://follower1:8080"]
	W     int      // write quorum size (including self)
	R     int      // read quorum size
	N     int      // total nodes (should be 5)
}

// ReplicationManager handles write propagation and quorum reads.
type ReplicationManager struct {
	Store  *store.KVStore
	Config Config
}

// ReplicateEntry is the JSON body the Leader sends to Followers.
type ReplicateEntry struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int64  `json:"version"`
}

// =====================================================
// Internal endpoints (called between nodes, not by client)
// =====================================================

// HandleInternalSet — POST /internal/set
// Called by Leader to push an update to this Follower.
func (rm *ReplicationManager) HandleInternalSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var entry ReplicateEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Assignment: Follower sleeps 100ms before applying the write
	time.Sleep(100 * time.Millisecond)

	rm.Store.SetWithVersion(entry.Key, entry.Value, entry.Version)
	w.WriteHeader(http.StatusOK)
}

// HandleInternalGet — GET /internal/get?key=xxx
// Called by Leader for quorum reads on this Follower.
func (rm *ReplicationManager) HandleInternalGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key cannot be empty", http.StatusBadRequest)
		return
	}

	// Assignment: Follower sleeps 50ms before responding to a read from Leader
	time.Sleep(50 * time.Millisecond)

	entry, exists := rm.Store.Get(key)
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

// =====================================================
// Leader write logic
// =====================================================

// LeaderSet handles a client write on the Leader.
//  1. Write to local store (self = 1 ack)
//  2. Replicate to followers, sleeping 200ms between each send
//  3. Return once W acks are collected (including self)
func (rm *ReplicationManager) LeaderSet(key, value string) (int64, error) {
	// Step 1: write locally
	version := rm.Store.Set(key, value)
	acksNeeded := rm.Config.W - 1 // self counts as 1

	if acksNeeded <= 0 {
		// W=1: return immediately, replicate in background
		go rm.replicateToAll(key, value, version)
		return version, nil
	}

	// W>1: send to followers and collect acks
	ackCh := make(chan bool, len(rm.Config.Peers))

	for _, peer := range rm.Config.Peers {
		go func(addr string) {
			err := rm.sendReplicate(addr, key, value, version)
			ackCh <- (err == nil)
		}(peer)
		// Assignment: Leader sleeps 200ms after sending to each Follower
		time.Sleep(200 * time.Millisecond)
	}

	// Wait for enough acks
	acks := 0
	for i := 0; i < len(rm.Config.Peers); i++ {
		if <-ackCh {
			acks++
		}
		if acks >= acksNeeded {
			return version, nil
		}
	}

	return version, fmt.Errorf("write quorum failed: got %d/%d", acks+1, rm.Config.W)
}

// replicateToAll sends updates to all peers asynchronously (used when W=1).
func (rm *ReplicationManager) replicateToAll(key, value string, version int64) {
	for _, peer := range rm.Config.Peers {
		if err := rm.sendReplicate(peer, key, value, version); err != nil {
			log.Printf("async replication to %s failed: %v", peer, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// sendReplicate sends a POST /internal/set to one peer.
func (rm *ReplicationManager) sendReplicate(addr, key, value string, version int64) error {
	body, _ := json.Marshal(ReplicateEntry{Key: key, Value: value, Version: version})

	resp, err := http.Post(addr+"/internal/set", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer %s returned %d", addr, resp.StatusCode)
	}
	return nil
}

// =====================================================
// Quorum read logic
// =====================================================

// QuorumRead reads from R nodes and returns the entry with the highest version.
func (rm *ReplicationManager) QuorumRead(key string) (store.KVEntry, bool) {
	type result struct {
		entry store.KVEntry
		found bool
	}

	var mu sync.Mutex
	var results []result

	// Local read (no sleep for the leader itself)
	localEntry, localFound := rm.Store.Get(key)
	results = append(results, result{localEntry, localFound})

	// R=1: just return local
	if rm.Config.R <= 1 {
		return localEntry, localFound
	}

	// Query peers in parallel
	resCh := make(chan result, len(rm.Config.Peers))
	for _, peer := range rm.Config.Peers {
		go func(addr string) {
			resp, err := http.Get(addr + "/internal/get?key=" + url.QueryEscape(key))
			if err != nil || resp.StatusCode != http.StatusOK {
				if resp != nil {
					resp.Body.Close()
				}
				resCh <- result{store.KVEntry{}, false}
				return
			}
			defer resp.Body.Close()

			var data struct {
				Value   string `json:"value"`
				Version int64  `json:"version"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				resCh <- result{store.KVEntry{}, false}
				return
			}
			resCh <- result{store.KVEntry{Value: data.Value, Version: data.Version}, true}
		}(peer)
	}

	// Collect R-1 more responses
	needed := rm.Config.R - 1
	collected := 0
	for i := 0; i < len(rm.Config.Peers); i++ {
		r := <-resCh
		mu.Lock()
		results = append(results, r)
		collected++
		mu.Unlock()
		if collected >= needed {
			break
		}
	}

	// Return the entry with the highest version
	var best store.KVEntry
	found := false
	for _, r := range results {
		if r.found && r.entry.Version > best.Version {
			best = r.entry
			found = true
		}
	}
	return best, found
}