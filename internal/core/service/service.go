package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"distributed-cache-service/internal/core/ports"
)

// ensure implementation
var _ ports.CacheService = (*ServiceImpl)(nil)

// ServiceImpl implements the CacheService interface.
// It orchestrates interactions between the storage (Read) and consensus (Write) layers.
type ServiceImpl struct {
	store     ports.Storage
	consensus ports.Consensus
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

type Command struct {
	Op    CommandType   `json:"op"`
	Key   string        `json:"key"`
	Value string        `json:"value,omitempty"`
	TTL   time.Duration `json:"ttl,omitempty"`
}

// Get retrieves a value from the local store.
// Note: This operation is eventually consistent (read-from-any).
// For strong consistency, it should be routed to the leader or use ReadIndex.
func (s *ServiceImpl) Get(ctx context.Context, key string) (string, error) {
	// For strong consistency, we could check if we are leader or read through consensus.
	// For high throughput/lower latency, we read from local store (eventual consistency if follower).
	val, found := s.store.Get(key)
	if !found {
		return "", fmt.Errorf("key not found")
	}
	return val, nil
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
