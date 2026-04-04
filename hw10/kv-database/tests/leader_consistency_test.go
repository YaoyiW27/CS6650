package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

const (
	leaderAddr    = "http://localhost:8080"
	follower1Addr = "http://localhost:8081"
	follower2Addr = "http://localhost:8082"
	follower3Addr = "http://localhost:8083"
	follower4Addr = "http://localhost:8084"
)

var followerAddrs = []string{follower1Addr, follower2Addr, follower3Addr, follower4Addr}

type kvResponse struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int64  `json:"version"`
}

// ============================================================
// Test 1: After Leader acknowledges write, read from Leader
//         should return consistent data.
// ============================================================
func TestLeader_ReadFromLeader_IsConsistent(t *testing.T) {
	key := fmt.Sprintf("test-leader-%d", time.Now().UnixNano())
	value := "consistent-value"

	// Write to leader and wait for ack
	version := mustSet(t, leaderAddr, key, value)
	t.Logf("Set key=%s value=%s version=%d on leader", key, value, version)

	// Read from leader — should be consistent
	got := mustGet(t, leaderAddr, key)
	if got.Value != value {
		t.Errorf("Leader read inconsistent: expected %q, got %q", value, got.Value)
	}
	t.Logf("Leader read: value=%s version=%d — consistent ✓", got.Value, got.Version)
}

// ============================================================
// Test 2: After Leader acknowledges write (W=5 means all
//         followers confirmed), read from any Follower via
//         /get should return consistent data.
// ============================================================
func TestLeader_ReadFromFollower_AfterAck_IsConsistent(t *testing.T) {
	key := fmt.Sprintf("test-follower-%d", time.Now().UnixNano())
	value := "replicated-value"

	version := mustSet(t, leaderAddr, key, value)
	t.Logf("Set key=%s version=%d on leader (W=5, all followers acked)", key, version)

	// Read from each follower — all should be consistent after W=5 ack
	for _, addr := range followerAddrs {
		got := mustGet(t, addr, key)
		if got.Value != value {
			t.Errorf("Follower %s read inconsistent: expected %q, got %q", addr, value, got.Value)
		} else {
			t.Logf("Follower %s read: value=%s version=%d — consistent ✓", addr, got.Value, got.Version)
		}
	}
}

// ============================================================
// Test 3: Use local_read during the update window to catch
//         inconsistency. We send a write to the leader and
//         IMMEDIATELY read local_read on followers.
//         With W=1 config, this should show stale data.
//
// NOTE: This test is most effective with W=1 (change docker-compose
//       to W=1, R=5 before running). With W=5 the leader waits
//       for all followers, so the window is tiny.
// ============================================================
func TestLeader_LocalRead_DuringUpdate_MayBeInconsistent(t *testing.T) {
	inconsistencyFound := false
	attempts := 50

	for i := 0; i < attempts; i++ {
		key := fmt.Sprintf("race-key-%d", i)
		initialValue := fmt.Sprintf("old-%d", i)
		newValue := fmt.Sprintf("new-%d", i)

		// First, write initial value and let it fully propagate
		mustSet(t, leaderAddr, key, initialValue)
		time.Sleep(1500 * time.Millisecond) // wait for full propagation

		// Now write a NEW value — and immediately try local_read on followers
		var wg sync.WaitGroup

		// Fire the write (don't wait for it)
		wg.Add(1)
		go func() {
			defer wg.Done()
			doSet(leaderAddr, key, newValue)
		}()

		// Immediately try local_read on all followers
		time.Sleep(10 * time.Millisecond) // tiny delay so the write starts
		for _, addr := range followerAddrs {
			resp, err := http.Get(addr + "/local_read?key=" + url.QueryEscape(key))
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			var got kvResponse
			json.NewDecoder(resp.Body).Decode(&got)

			if got.Value == initialValue {
				t.Logf("INCONSISTENCY DETECTED on %s: key=%s expected=%q got=%q (stale!)",
					addr, key, newValue, got.Value)
				inconsistencyFound = true
			}
		}
		wg.Wait()

		if inconsistencyFound {
			break
		}
	}

	if inconsistencyFound {
		t.Log("Successfully demonstrated inconsistency window via local_read ✓")
	} else {
		t.Log("No inconsistency detected in this run (may need W=1 config or more attempts)")
	}
}

// ============================================================
// Helpers
// ============================================================

func mustSet(t *testing.T, addr, key, value string) int64 {
	t.Helper()
	version, err := doSet(addr, key, value)
	if err != nil {
		t.Fatalf("Failed to SET %s=%s on %s: %v", key, value, addr, err)
	}
	return version
}

func doSet(addr, key, value string) (int64, error) {
	reqURL := fmt.Sprintf("%s/set?key=%s&value=%s", addr, url.QueryEscape(key), url.QueryEscape(value))
	req, _ := http.NewRequest(http.MethodPut, reqURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var result kvResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Version, nil
}

func mustGet(t *testing.T, addr, key string) kvResponse {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("%s/get?key=%s", addr, url.QueryEscape(key)))
	if err != nil {
		t.Fatalf("Failed to GET key=%s from %s: %v", key, addr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET key=%s from %s: expected 200, got %d: %s", key, addr, resp.StatusCode, body)
	}

	var result kvResponse
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}