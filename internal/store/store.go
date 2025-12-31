package store

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"distributed-cache-service/internal/store/policy"
)

// Item represents a single cached value with its metadata.
type Item struct {
	Value      string `json:"value"`
	Expiration int64  `json:"expiration"` // Unix timestamp in nanoseconds when this item expires. 0 means no expiration.
}

// Store implements a thread-safe in-memory key-value cache.
// It supports TTL-based expiration and basic CRUD operations.
type Store struct {
	mu       sync.RWMutex
	items    map[string]*Item
	capacity int
	policy   policy.EvictionPolicy
}

// Option defines a functional option for configuring the store.
type Option func(*Store)

// WithCapacity sets the maximum number of items in the store.
func WithCapacity(capacity int) Option {
	return func(s *Store) {
		s.capacity = capacity
	}
}

// WithPolicy sets the eviction policy.
func WithPolicy(p policy.EvictionPolicy) Option {
	return func(s *Store) {
		s.policy = p
	}
}

// New creates a new, empty Store instance with optional configuration.
// Default capacity is 0 (unlimited) and policy is nil (no eviction).
func New(opts ...Option) *Store {
	s := &Store{
		items:    make(map[string]*Item),
		capacity: 0,               // Default unlimited
		policy:   policy.NewLRU(), // Default LRU if capacity set? Or just nil.
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Get retrieves the value associated with the given key.
// It returns the value and true if the key exists and has not expired.
// If the key is not found or has expired, it returns an empty string and false.
func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock() // Lock for policy update
	defer s.mu.Unlock()

	item, found := s.items[key]
	if !found {
		return "", false
	}

	if item.Expiration > 0 && time.Now().UnixNano() > item.Expiration {
		// Lazy deletion? Or just return not found.
		// If we return not found, we should probably delete it or let cleanup handle it.
		// Policy OnAccess should probably NOT be called if expired.
		return "", false
	}

	if s.policy != nil {
		s.policy.OnAccess(key)
	}

	return item.Value, true
}

// Set adds or updates a key with the provided value and Time-To-Live (TTL).
// If ttl is 0, the item will never expire.
func (s *Store) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if update
	if _, exists := s.items[key]; exists {
		if s.policy != nil {
			s.policy.OnAccess(key)
		}
	} else {
		// New item
		// Evict if full
		if s.capacity > 0 && len(s.items) >= s.capacity && s.policy != nil {
			victim := s.policy.SelectVictim()
			if victim != "" {
				s.deleteInternal(victim)
			}
		}
		if s.policy != nil {
			s.policy.OnAdd(key)
		}
	}

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
	s.deleteInternal(key)
}

func (s *Store) deleteInternal(key string) {
	if _, exists := s.items[key]; exists {
		delete(s.items, key)
		if s.policy != nil {
			s.policy.OnRemove(key)
		}
	}
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
