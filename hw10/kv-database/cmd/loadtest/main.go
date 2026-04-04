package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================
// Config
// ============================================================

var (
	writeAddr   = flag.String("write-addr", "http://localhost:8080", "address to send writes (leader or any node)")
	readAddrs   = flag.String("read-addrs", "http://localhost:8080,http://localhost:8081,http://localhost:8082,http://localhost:8083,http://localhost:8084", "comma-separated read addresses")
	writePct    = flag.Int("write-pct", 10, "percentage of requests that are writes (1,10,50,90)")
	totalReqs   = flag.Int("total", 2000, "total number of requests to send")
	concurrency = flag.Int("concurrency", 20, "number of concurrent workers")
	numKeys     = flag.Int("keys", 10, "number of distinct keys (smaller = more temporal locality)")
	outputFile  = flag.String("output", "results.csv", "output CSV file")
	dbType      = flag.String("db-type", "leader-w5r1", "label for this config (leader-w5r1, leader-w1r5, leader-w3r3, leaderless)")
)

// ============================================================
// Tracking
// ============================================================

type RequestResult struct {
	Timestamp   time.Time
	Type        string // "read" or "write"
	Key         string
	LatencyMs   float64
	Version     int64
	StaleRead   bool
	DBType      string
	WritePct    int
}

// latestVersion tracks the latest known version per key (from writes).
// Used to detect stale reads.
type versionTracker struct {
	mu       sync.RWMutex
	versions map[string]int64
}

func (vt *versionTracker) update(key string, version int64) {
	vt.mu.Lock()
	defer vt.mu.Unlock()
	if version > vt.versions[key] {
		vt.versions[key] = version
	}
}

func (vt *versionTracker) get(key string) int64 {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.versions[key]
}

// writeTimestamps tracks the last write time per key for interval measurement.
type writeTimestampTracker struct {
	mu    sync.RWMutex
	times map[string]time.Time
}

func (wt *writeTimestampTracker) record(key string, t time.Time) {
	wt.mu.Lock()
	defer wt.mu.Unlock()
	wt.times[key] = t
}

func (wt *writeTimestampTracker) get(key string) (time.Time, bool) {
	wt.mu.RLock()
	defer wt.mu.RUnlock()
	t, ok := wt.times[key]
	return t, ok
}

// ============================================================
// Main
// ============================================================

func main() {
	flag.Parse()

	reads := parseAddrs(*readAddrs)
	fmt.Printf("Load Test: db=%s write-pct=%d%% total=%d concurrency=%d keys=%d\n",
		*dbType, *writePct, *totalReqs, *concurrency, *numKeys)
	fmt.Printf("  Write addr: %s\n", *writeAddr)
	fmt.Printf("  Read addrs: %v\n", reads)

	vt := &versionTracker{versions: make(map[string]int64)}
	wt := &writeTimestampTracker{times: make(map[string]time.Time)}

	var results []RequestResult
	var resultsMu sync.Mutex
	var staleCount atomic.Int64

	// Pre-generate the request plan
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	type request struct {
		isWrite bool
		key     string
	}
	plan := make([]request, *totalReqs)
	for i := range plan {
		key := fmt.Sprintf("key-%d", rng.Intn(*numKeys))
		isWrite := rng.Intn(100) < *writePct
		plan[i] = request{isWrite: isWrite, key: key}
	}

	// Execute with worker pool
	work := make(chan request, *totalReqs)
	var wg sync.WaitGroup

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localRng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(rng.Intn(10000))))
			client := &http.Client{Timeout: 10 * time.Second}

			for req := range work {
				var result RequestResult
				result.Key = req.key
				result.DBType = *dbType
				result.WritePct = *writePct
				result.Timestamp = time.Now()

				if req.isWrite {
					result.Type = "write"
					start := time.Now()
					version, err := doWrite(client, *writeAddr, req.key)
					result.LatencyMs = float64(time.Since(start).Microseconds()) / 1000.0

					if err == nil {
						result.Version = version
						vt.update(req.key, version)
						wt.record(req.key, start)
					}
				} else {
					result.Type = "read"
					// Pick a random read address
					addr := reads[localRng.Intn(len(reads))]
					start := time.Now()
					value, version, err := doRead(client, addr, req.key)
					result.LatencyMs = float64(time.Since(start).Microseconds()) / 1000.0

					if err == nil {
						result.Version = version
						latest := vt.get(req.key)
						if version < latest {
							result.StaleRead = true
							staleCount.Add(1)
						}
						_ = value

						// Record read-write interval
						if wTime, ok := wt.get(req.key); ok {
							_ = start.Sub(wTime) // interval captured in CSV via timestamp
						}
					}
				}

				resultsMu.Lock()
				results = append(results, result)
				resultsMu.Unlock()
			}
		}()
	}

	// Feed work
	startTime := time.Now()
	for _, req := range plan {
		work <- req
	}
	close(work)
	wg.Wait()
	elapsed := time.Since(startTime)

	// Summary
	var readCount, writeCount int
	var readLatSum, writeLatSum float64
	for _, r := range results {
		if r.Type == "read" {
			readCount++
			readLatSum += r.LatencyMs
		} else {
			writeCount++
			writeLatSum += r.LatencyMs
		}
	}

	fmt.Printf("\n--- Results ---\n")
	fmt.Printf("Total time: %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Reads:  %d (avg %.1fms)\n", readCount, readLatSum/max(float64(readCount), 1))
	fmt.Printf("Writes: %d (avg %.1fms)\n", writeCount, writeLatSum/max(float64(writeCount), 1))
	fmt.Printf("Stale reads: %d / %d (%.1f%%)\n",
		staleCount.Load(), readCount,
		float64(staleCount.Load())/max(float64(readCount), 1)*100)

	// Write CSV
	writeCSV(*outputFile, results)
	fmt.Printf("Results written to %s\n", *outputFile)
}

// ============================================================
// HTTP helpers
// ============================================================

func doWrite(client *http.Client, addr, key string) (int64, error) {
	value := fmt.Sprintf("v-%d", time.Now().UnixNano())
	reqURL := fmt.Sprintf("%s/set?key=%s&value=%s", addr, url.QueryEscape(key), url.QueryEscape(value))
	req, _ := http.NewRequest(http.MethodPut, reqURL, nil)
	resp, err := client.Do(req)
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

func doRead(client *http.Client, addr, key string) (string, int64, error) {
	reqURL := fmt.Sprintf("%s/get?key=%s", addr, url.QueryEscape(key))
	resp, err := client.Get(reqURL)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", 0, fmt.Errorf("not found")
	}

	var result struct {
		Value   string `json:"value"`
		Version int64  `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Value, result.Version, nil
}

// ============================================================
// CSV output
// ============================================================

func writeCSV(filename string, results []RequestResult) {
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating CSV: %v\n", err)
		return
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"timestamp", "type", "key", "latency_ms", "version", "stale_read", "db_type", "write_pct"})

	for _, r := range results {
		w.Write([]string{
			r.Timestamp.Format(time.RFC3339Nano),
			r.Type,
			r.Key,
			fmt.Sprintf("%.3f", r.LatencyMs),
			strconv.FormatInt(r.Version, 10),
			strconv.FormatBool(r.StaleRead),
			r.DBType,
			strconv.Itoa(r.WritePct),
		})
	}
}

// ============================================================
// Helpers
// ============================================================

func parseAddrs(s string) []string {
	var addrs []string
	for _, a := range splitComma(s) {
		if a != "" {
			addrs = append(addrs, a)
		}
	}
	return addrs
}

func splitComma(s string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}