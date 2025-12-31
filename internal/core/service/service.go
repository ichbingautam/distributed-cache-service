package service

import (
	"context"
	"distributed-cache-service/internal/core/ports"
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
	store     ports.Storage
	consensus ports.Consensus
	// requestGroup handles single-flight request coalescing for hot keys.
	requestGroup singleflight.Group
}

// New creates a new instance of the cache service.
func New(store ports.Storage, consensus ports.Consensus) *ServiceImpl {
	return &ServiceImpl{
		store:     store,
		consensus: consensus,
	}
}

// Command definitions shared with Raft FSM
type CommandType string

const (
	SetOp    CommandType = "SET"
	DeleteOp CommandType = "DELETE"
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
// Consistency Level: Eventual (Read-From-Any).
// - Calls to Get do not go through Raft; they read directly from the node's local FSM state.
// - If strong consistency is required, a ReadIndex or Leader-process mechanism is needed (future work).
//
// Concurrency:
// - Uses SingleFlight to prevent cache stampedes (Thundering Herd).
// - Multiple concurrent requests for the same key are coalesced into a single lookup.
func (s *ServiceImpl) Get(ctx context.Context, key string) (string, error) {
	// Use SingleFlight to coalesce concurrent requests for the same key
	v, err, _ := s.requestGroup.Do(key, func() (interface{}, error) {
		val, found := s.store.Get(key)
		if !found {
			return "", fmt.Errorf("key not found")
		}
		return val, nil
	})

	if err != nil {
		return "", err
	}

	return v.(string), nil
}

// Set replicates a Set command via the consensus layer.
// This operation is strongly consistent (linearizable) as it goes through Raft log.
func (s *ServiceImpl) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	cmd := Command{
		Op:    SetOp,
		Key:   key,
		Value: value,
		TTL:   ttl,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	return s.consensus.Apply(data)
}

// Delete replicates a Delete command via the consensus layer.
func (s *ServiceImpl) Delete(ctx context.Context, key string) error {
	cmd := Command{
		Op:  DeleteOp,
		Key: key,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	return s.consensus.Apply(data)
}

// Join adds a new node to the cluster by invoking the consensus layer.
func (s *ServiceImpl) Join(ctx context.Context, nodeID, addr string) error {
	return s.consensus.AddVoter(nodeID, addr)
}
