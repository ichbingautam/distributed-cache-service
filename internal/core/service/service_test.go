// Package service_test contains unit tests for the service layer.
package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

// MockStore implements ports.Storage for testing.
// It simulates thread-safe storage operations and basic latency.
type MockStore struct {
	mu    sync.Mutex
	data  map[string]string
	calls int
}

func (m *MockStore) Get(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	// Simulate slow DB/IO
	time.Sleep(10 * time.Millisecond)
	val, ok := m.data[key]
	return val, ok
}

func (m *MockStore) Set(key, value string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *MockStore) Delete(key string) {}

// MockConsensus implements ports.Consensus for testing.
// It serves as a no-op stub for consensus operations unless extended.
type MockConsensus struct{}

func (m *MockConsensus) Apply(cmd []byte) error         { return nil }
func (m *MockConsensus) AddVoter(id, addr string) error { return nil }
func (m *MockConsensus) IsLeader() bool                 { return true }
func (m *MockConsensus) VerifyLeader() error            { return nil }

func TestService_Get_Concurrency(t *testing.T) {
	mockStore := &MockStore{
		data: map[string]string{"key1": "value1"},
	}
	mockConsensus := &MockConsensus{}
	svc := New(mockStore, mockConsensus)

	ctx := context.Background()
	concurrency := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Launch concurrent requests
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			val, err := svc.Get(ctx, "key1")
			if err != nil {
				t.Errorf("Get failed: %v", err)
			}
			if val != "value1" {
				t.Errorf("Expected value1, got %s", val)
			}
		}()
	}

	wg.Wait()

	// Verify that the store was essentially hit fewer times than requests.
	// In a perfect singleflight scenarios with 100 requests arriving at exact same time, it should be 1 call.
	// But due to scheduling, it might be slightly more, but definitely < 100.
	mockStore.mu.Lock()
	calls := mockStore.calls
	mockStore.mu.Unlock()

	t.Logf("Total requests: %d, Actual Store calls: %d", concurrency, calls)

	if calls > 20 { // Arbitrary threshold, typically should be close to 1
		t.Errorf("Significantly failed to coalesce requests. Calls: %d", calls)
	}
}
