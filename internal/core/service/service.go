package service

import (
	"context"
	"distributed-cache-service/internal/core/ports"
	"distributed-cache-service/internal/observability"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

// ensure implementation
var _ ports.CacheService = (*ServiceImpl)(nil)

// ServiceImpl implements the CacheService interface.
// It orchestrates interactions between the storage (Read) and consensus (Write) layers.
// It manages data consistency and request concurrency.
type ServiceImpl struct {
	store        ports.Storage
	consensus    ports.Consensus
	requestGroup singleflight.Group
	consistency  ConsistencyMode
}

// New creates a new instance of the cache service.
func New(store ports.Storage, consensus ports.Consensus, consistency ConsistencyMode) *ServiceImpl {
	return &ServiceImpl{
		store:       store,
		consensus:   consensus,
		consistency: consistency,
	}
}

// Command definitions shared with Raft FSM
type CommandType string

const (
	SetOp    CommandType = "SET"
	DeleteOp CommandType = "DELETE"
)

// ConsistencyMode defines the consistency level for read operations.
type ConsistencyMode string

const (
	ConsistencyStrong   ConsistencyMode = "strong"
	ConsistencyEventual ConsistencyMode = "eventual"
)

// Command represents a state machine command to be replicated via Raft.
type Command struct {
	Op    CommandType   `json:"op"`
	Key   string        `json:"key"`
	Value string        `json:"value,omitempty"`
	TTL   time.Duration `json:"ttl,omitempty"`
}

// Get retrieves a value from the local store.
//
// Consistency Level: Tunable (Strong vs Eventual).
// - Strong: Verifies leadership (Linearizable).
// - Eventual: Reads local state immediately.
//
// Concurrency:
// - Uses SingleFlight to prevent cache stampedes (Thundering Herd).
// - Multiple concurrent requests for the same key are coalesced into a single lookup.
func (s *ServiceImpl) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()

	// Ensure Strong Consistency: Check if we are still the leader
	if s.consistency == ConsistencyStrong {
		if err := s.consensus.VerifyLeader(); err != nil {
			observability.CacheOperationsTotal.WithLabelValues("get", "error").Inc()
			return "", fmt.Errorf("consistency check failed: %w", err)
		}
	}

	// Use SingleFlight to coalesce concurrent requests for the same key
	v, err, _ := s.requestGroup.Do(key, func() (interface{}, error) {
		val, found := s.store.Get(key)
		if !found {
			observability.CacheMissesTotal.Inc()
			observability.CacheOperationsTotal.WithLabelValues("get", "miss").Inc()
			return "", fmt.Errorf("key not found")
		}
		observability.CacheHitsTotal.Inc()
		observability.CacheOperationsTotal.WithLabelValues("get", "hit").Inc()
		return val, nil
	})
	observability.CacheDurationSeconds.WithLabelValues("get").Observe(time.Since(start).Seconds())

	if err != nil {
		return "", err
	}

	return v.(string), nil
}

// Set stores a value in the system (Strongly Consistent via Raft).
func (s *ServiceImpl) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		observability.CacheDurationSeconds.WithLabelValues("set").Observe(time.Since(start).Seconds())
	}()

	cmd := Command{
		Op:    SetOp,
		Key:   key,
		Value: value,
		TTL:   ttl,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		observability.CacheOperationsTotal.WithLabelValues("set", "error").Inc()
		return err
	}

	if err := s.consensus.Apply(data); err != nil {
		observability.CacheOperationsTotal.WithLabelValues("set", "error").Inc()
		return err
	}
	observability.CacheOperationsTotal.WithLabelValues("set", "success").Inc()
	return nil
}

// Delete removes a value from the system (Strongly Consistent via Raft).
func (s *ServiceImpl) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		observability.CacheDurationSeconds.WithLabelValues("delete").Observe(time.Since(start).Seconds())
	}()

	cmd := Command{
		Op:  DeleteOp,
		Key: key,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		observability.CacheOperationsTotal.WithLabelValues("delete", "error").Inc()
		return err
	}

	if err := s.consensus.Apply(data); err != nil {
		observability.CacheOperationsTotal.WithLabelValues("delete", "error").Inc()
		return err
	}
	observability.CacheOperationsTotal.WithLabelValues("delete", "success").Inc()
	return nil
}

// Join adds a new node to the cluster by invoking the consensus layer.
func (s *ServiceImpl) Join(ctx context.Context, nodeID, addr string) error {
	return s.consensus.AddVoter(nodeID, addr)
}
