package store

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// Item represents a single cached value with its metadata.
type Item struct {
	Value      string `json:"value"`
	Expiration int64  `json:"expiration"` // Unix timestamp in nanoseconds when this item expires. 0 means no expiration.
}

// Store implements a thread-safe in-memory key-value cache.
// It supports TTL-based expiration and basic CRUD operations.
type Store struct {
	mu    sync.RWMutex
	items map[string]*Item
}

// New creates a new, empty Store instance.
func New() *Store {
	return &Store{
		items: make(map[string]*Item),
	}
}

// Get retrieves the value associated with the given key.
// It returns the value and true if the key exists and has not expired.
// If the key is not found or has expired, it returns an empty string and false.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, found := s.items[key]
	if !found {
		return "", false
	}

	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		return "", false
	}

	return item.Value, true
}

// Set adds or updates a key with the provided value and Time-To-Live (TTL).
// If ttl is 0, the item will never expire.
func (s *Store) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiration := int64(0)
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	s.items[key] = &Item{
		Value:      value,
		Expiration: expiration,
	}
}

// Delete removes the item associated with the given key from the store.
// If the key does not exist, this is a no-op.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

// StartCleanup starts a background goroutine that periodically removes expired items.
// The cleanup runs at the specified interval.
// Note: This function spawns a goroutine and does not provide a way to stop it in this simple implementation.
func (s *Store) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.deleteExpired()
		}
	}()
}

func (s *Store) deleteExpired() {
	now := time.Now().UnixNano()
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range s.items {
		if v.Expiration > 0 && now > v.Expiration {
			delete(s.items, k)
		}
	}
}

// Snapshot serializes the current state of the store to the provided writer (IO sink).
// This is used by Raft to take snapshots of the state machine.
func (s *Store) Snapshot(w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.NewEncoder(w).Encode(s.items)
}

// Restore replaces the current state of the store with the data read from the provided reader.
// This is used by Raft to restore the state machine from a snapshot.
func (s *Store) Restore(r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.NewDecoder(r).Decode(&s.items)
}
