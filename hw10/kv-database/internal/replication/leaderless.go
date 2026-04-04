package replication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// CoordinatorSet handles a client write on any leaderless node.
// This node becomes the Write Coordinator:
//  1. Write locally (self = 1 ack)
//  2. Forward to all other nodes via /internal/set, sleep 200ms between each
//  3. Wait for ALL to ack (W=N), then return 201
func (rm *ReplicationManager) CoordinatorSet(key, value string) (int64, error) {
	// Step 1: write locally
	version := rm.Store.Set(key, value)

	// Step 2: replicate to all peers
	ackCh := make(chan error, len(rm.Config.Peers))

	for _, peer := range rm.Config.Peers {
		go func(addr string) {
			ackCh <- rm.sendReplicate(addr, key, value, version)
		}(peer)
		// Assignment: Coordinator sleeps 200ms after sending to each node
		time.Sleep(200 * time.Millisecond)
	}

	// Step 3: W=N, so we need ALL peers to ack
	failures := 0
	for i := 0; i < len(rm.Config.Peers); i++ {
		if err := <-ackCh; err != nil {
			log.Printf("replication to peer failed: %v", err)
			failures++
		}
	}

	if failures > 0 {
		return version, fmt.Errorf("write coordinator: %d/%d peers failed", failures, len(rm.Config.Peers))
	}
	return version, nil
}

// sendReplicate is already defined in leader.go — reused here.

// HandleInternalSetLeaderless is the same as HandleInternalSet.
// Follower sleeps 100ms before applying. Reuses the same /internal/set endpoint.
// (No new handler needed — the existing HandleInternalSet works for both modes.)

// ForwardWrite is used when a leaderless node receives a write but we want
// explicit coordinator behavior. For the leaderless W=N case, we use CoordinatorSet.

// LeaderlessGet just reads locally (R=1).
func (rm *ReplicationManager) LeaderlessGet(key string) (map[string]interface{}, bool) {
	entry, exists := rm.Store.Get(key)
	if !exists {
		return nil, false
	}
	return map[string]interface{}{
		"value":   entry.Value,
		"version": entry.Version,
	}, true
}

// sendReplicateLeaderless sends a POST /internal/set to one peer.
// This is identical to sendReplicate in leader.go — we reuse that function.
// Keeping this comment here for clarity: leaderless replication uses the
// exact same internal endpoint and mechanism as leader-follower.

// Placeholder: if you need a leaderless-specific forwarding mechanism
// (e.g., forwarding the original client request), add it here.

// ForwardSetToCoordinator can be used if a client accidentally sends a write
// to a node that should delegate. For our leaderless design, every node IS
// a valid coordinator, so this isn't needed — but kept as documentation.
func ForwardSetToCoordinator(coordinatorAddr, key, value string) (int64, error) {
	body, _ := json.Marshal(map[string]string{"key": key, "value": value})
	resp, err := http.Post(coordinatorAddr+"/set", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Version int64 `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Version, nil
}