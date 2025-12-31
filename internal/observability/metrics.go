package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CacheOperationsTotal counts get/set/delete operations
	CacheOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_operations_total",
		Help: "The total number of cache operations",
	}, []string{"type", "status"})

	// CacheHitsTotal counts cache hits
	CacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "The total number of cache hits",
	})

	// CacheMissesTotal counts cache misses
	CacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "The total number of cache misses",
	})

	// CacheDurationSeconds measures latency
	CacheDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cache_duration_seconds",
		Help:    "The latency of cache operations",
		Buckets: prometheus.DefBuckets,
	}, []string{"type"})
)
