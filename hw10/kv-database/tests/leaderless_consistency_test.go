package tests

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

// Leaderless nodes are all on ports 8080-8084
var leaderlessNodes = []string{
	"http://localhost:8080",
	"http://localhost:8081",
	"http://localhost:8082",
	"http://localhost:8083",
	"http://localhost:8084",
}

// ============================================================
// Test 1: Write to a random node (Write Coordinator), then
//         immediately read from OTHER nodes during the update
//         window. Should catch inconsistency since R=1 reads
//         local data that may not yet be updated.
// ============================================================
func TestLeaderless_InconsistencyWindow(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	inconsistencyFound := false
	attempts := 50

	for i := 0; i < attempts; i++ {
		key := fmt.Sprintf("ll-race-%d", i)
		initialValue := fmt.Sprintf("init-%d", i)
		newValue := fmt.Sprintf("updated-%d", i)

		// Pick a random coordinator for the initial write
		coord := rng.Intn(len(leaderlessNodes))
		mustSet(t, leaderlessNodes[coord], key, initialValue)
		time.Sleep(1500 * time.Millisecond) // let it fully propagate

		// Pick a DIFFERENT coordinator for the new write
		newCoord := (coord + 1) % len(leaderlessNodes)

		var wg sync.WaitGroup

		// Fire the write asynchronously
		wg.Add(1)
		go func() {
			defer wg.Done()
			doSet(leaderlessNodes[newCoord], key, newValue)
		}()

		// Immediately read from other nodes (not the coordinator)
		time.Sleep(10 * time.Millisecond)
		for j, addr := range leaderlessNodes {
			if j == newCoord {
				continue // skip the coordinator
			}

			resp, err := http.Get(addr + "/get?key=" + url.QueryEscape(key))
			if err != nil {
				continue
			}

			var got kvResponse
			json.NewDecoder(resp.Body).Decode(&got)
			resp.Body.Close()

			if got.Value == initialValue {
				t.Logf("INCONSISTENCY on node %d (%s): key=%s expected=%q got=%q (stale!)",
					j, addr, key, newValue, got.Value)
				inconsistencyFound = true
			}
		}
		wg.Wait()

		if inconsistencyFound {
			break
		}
	}

	if inconsistencyFound {
		t.Log("Successfully demonstrated leaderless inconsistency window ✓")
	} else {
		t.Log("No inconsistency detected (timing-dependent — try increasing attempts)")
	}
}

// ============================================================
// Test 2: After Coordinator acknowledges write, read from
//         the Coordinator itself — should be consistent.
// ============================================================
func TestLeaderless_ReadFromCoordinator_AfterAck_IsConsistent(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("ll-coord-%d", i)
		value := fmt.Sprintf("val-%d", i)
		coord := rng.Intn(len(leaderlessNodes))

		// Write and wait for ack (W=N, so all nodes confirmed)
		version := mustSet(t, leaderlessNodes[coord], key, value)

		// Read from coordinator — must be consistent
		got := mustGet(t, leaderlessNodes[coord], key)
		if got.Value != value {
			t.Errorf("Coordinator %d inconsistent: expected %q got %q", coord, value, got.Value)
		} else {
			t.Logf("Coordinator node%d: key=%s value=%s version=%d — consistent ✓",
				coord, key, got.Value, version)
		}
	}
}

// ============================================================
// Test 3: After Coordinator acknowledges write (W=N means ALL
//         nodes confirmed), read from another node — should
//         be consistent.
// ============================================================
func TestLeaderless_ReadFromOtherNode_AfterAck_IsConsistent(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("ll-other-%d", i)
		value := fmt.Sprintf("val-%d", i)
		coord := rng.Intn(len(leaderlessNodes))
		other := (coord + 1) % len(leaderlessNodes)

		// Write — W=N so all nodes are updated before ack
		mustSet(t, leaderlessNodes[coord], key, value)

		// Read from a different node
		got := mustGet(t, leaderlessNodes[other], key)
		if got.Value != value {
			t.Errorf("Node %d inconsistent after W=N ack: expected %q got %q", other, value, got.Value)
		} else {
			t.Logf("Node%d (non-coordinator): key=%s value=%s — consistent ✓", other, key, got.Value)
		}
	}
}