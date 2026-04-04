package store

import (
	"sync"
)

// KVEntry holds a value and its logical version number.
type KVEntry struct {
	Value   string `json:"value"`
	Version int64  `json:"version"`
}

// KVStore is a thread-safe in-memory key-value store.
type KVStore struct {
	mu   sync.RWMutex
	data map[string]KVEntry
}

// NewKVStore creates a new empty KVStore.
func NewKVStore() *KVStore {
	return &KVStore{
		data: make(map[string]KVEntry),
	}
}

// Set stores the value under the given key, incrementing the version.
// Returns the new version number.
func (s *KVStore) Set(key, value string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.data[key]
	var newVersion int64
	if exists {
		newVersion = entry.Version + 1
	} else {
		newVersion = 1
	}

	s.data[key] = KVEntry{
		Value:   value,
		Version: newVersion,
	}
	return newVersion
}

// SetWithVersion stores the value with a specific version (used by followers
// receiving replicated data from the leader or coordinator).
func (s *KVStore) SetWithVersion(key, value string, version int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only apply if the incoming version is newer than what we have.
	if existing, exists := s.data[key]; exists && existing.Version >= version {
		return
	}
	s.data[key] = KVEntry{
		Value:   value,
		Version: version,
	}
}

// Get retrieves the entry for the given key.
// Returns the entry and true if found, or zero-value and false if not.
func (s *KVStore) Get(key string) (KVEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.data[key]
	return entry, exists
}