package service

import (
	"context"
	"distributed-cache-service/internal/observability"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// Re-use mocks from service_test.go.

func TestMetrics_Get(t *testing.T) {
	// Reset counters if needed? Prometheus globals are sticky.
	// In unit tests, usually we just check delta or strictly set value if we could reset.
	// For simplicity, we capture value before and after.

	// 1. Setup Mock from service_test.go
	// MockStore in service_test.go has fields: mu, data, calls
	mockStore := &MockStore{
		data: make(map[string]string),
	}
	mockConsensus := &MockConsensus{}
	svc := New(mockStore, mockConsensus)
	ctx := context.Background()

	// 2. Test Hit
	// We need to set data in a thread-safe way or just init it since we are the only user here
	mockStore.data["hit_key"] = "value"

	initialHits := testutil.ToFloat64(observability.CacheHitsTotal)

	val, err := svc.Get(ctx, "hit_key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	newHits := testutil.ToFloat64(observability.CacheHitsTotal)
	assert.Equal(t, initialHits+1, newHits, "CacheHitsTotal should increment by 1")

	// 3. Test Miss
	// key "miss_key" is not in data map, so it returns not found

	initialMisses := testutil.ToFloat64(observability.CacheMissesTotal)

	_, err = svc.Get(ctx, "miss_key")
	assert.Error(t, err)

	newMisses := testutil.ToFloat64(observability.CacheMissesTotal)
	assert.Equal(t, initialMisses+1, newMisses, "CacheMissesTotal should increment by 1")
}

func TestMetrics_Set(t *testing.T) {
	mockStore := &MockStore{
		data: make(map[string]string),
	}
	mockConsensus := &MockConsensus{}
	svc := New(mockStore, mockConsensus)
	ctx := context.Background()

	// MockConsensus.Apply in service_test.go returns nil, which is what we want (success).

	// We need to check Vector counters. testutil.ToFloat64 handles simple collectors.
	// For Vectors, we must get the specific metric instance with labels.
	// Note: If the metric with these labels hasn't been created/observed yet, it might return 0.
	// The service implementation calls WithLabelValues(...).Inc(), which initializes it.

	ctr := observability.CacheOperationsTotal.WithLabelValues("set", "success")
	initialSets := testutil.ToFloat64(ctr)

	err := svc.Set(ctx, "key", "val", time.Second)
	assert.NoError(t, err)

	newSets := testutil.ToFloat64(ctr)
	assert.Equal(t, initialSets+1, newSets, "CacheOperationsTotal(set, success) should increment")
}

func TestMetrics_Delete(t *testing.T) {
	mockStore := &MockStore{
		data: make(map[string]string),
	}
	mockConsensus := &MockConsensus{}
	svc := New(mockStore, mockConsensus)
	ctx := context.Background()

	ctr := observability.CacheOperationsTotal.WithLabelValues("delete", "success")
	initialDels := testutil.ToFloat64(ctr)

	err := svc.Delete(ctx, "key")
	assert.NoError(t, err)

	newDels := testutil.ToFloat64(ctr)
	assert.Equal(t, initialDels+1, newDels, "CacheOperationsTotal(delete, success) should increment")
}
