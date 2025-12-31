package store

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// Item represents a cached item
type Item struct {
	Value      string `json:"value"`
	Expiration int64  `json:"expiration"` // Unix timestamp in nanoseconds
}

// Store is a thread-safe in-memory cache
type Store struct {
	mu    sync.RWMutex
	items map[string]*Item
}

// New creates a new Store
func New() *Store {
	return &Store{
		items: make(map[string]*Item),
	}
}

// Get returns the value for a key and whether it was found
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

// Set adds or updates a key with a value and TTL
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

// Delete removes a key
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

// StartCleanup starts a background goroutine to remove expired items
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

// Snapshot writes the entire store to w
func (s *Store) Snapshot(w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.NewEncoder(w).Encode(s.items)
}

// Restore reads the store from r
func (s *Store) Restore(r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.NewDecoder(r).Decode(&s.items)
}
