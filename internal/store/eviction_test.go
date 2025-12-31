package store

import (
	"testing"

	"distributed-cache-service/internal/store/policy"

	"github.com/stretchr/testify/assert"
)

func TestStore_LRUEviction(t *testing.T) {
	// Capacity 2, LRU Policy
	s := New(WithCapacity(2), WithPolicy(policy.NewLRU()))

	// 1. Fill store
	s.Set("key1", "val1", 0)
	s.Set("key2", "val2", 0)

	// 2. Access key1 (making key2 the LRU)
	val, found := s.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "val1", val)

	// 3. Add key3 -> Should evict key2 (LRU)
	s.Set("key3", "val3", 0)

	// 4. Verify key2 is gone
	_, found = s.Get("key2")
	assert.False(t, found, "key2 should be evicted")

	// 5. Verify key1 and key3 are present
	_, found = s.Get("key1")
	assert.True(t, found)
	_, found = s.Get("key3")
	assert.True(t, found)
}

func TestStore_FIFOEviction(t *testing.T) {
	// Capacity 2, FIFO Policy
	s := New(WithCapacity(2), WithPolicy(policy.NewFIFO()))

	// 1. Add key1, key2
	s.Set("key1", "val1", 0)
	s.Set("key2", "val2", 0)

	// 2. Access key1 (FIFO usage ignores access)
	s.Get("key1")

	// 3. Add key3 -> Should evict key1 (First In)
	s.Set("key3", "val3", 0)

	// 4. Verify key1 is gone
	_, found := s.Get("key1")
	assert.False(t, found, "key1 should be evicted (FIFO)")

	// 5. Verify key2 (newer) and key3 (newest) exist
	_, found = s.Get("key2")
	assert.True(t, found)
	_, found = s.Get("key3")
	assert.True(t, found)
}
