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

type ServiceImpl struct {
	store     ports.Storage
	consensus ports.Consensus
}

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

func (s *ServiceImpl) Get(ctx context.Context, key string) (string, error) {
	// For strong consistency, we could check if we are leader or read through consensus.
	// For high throughput/lower latency, we read from local store (eventual consistency if follower).
	val, found := s.store.Get(key)
	if !found {
		return "", fmt.Errorf("key not found")
	}
	return val, nil
}

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

func (s *ServiceImpl) Join(ctx context.Context, nodeID, addr string) error {
	return s.consensus.AddVoter(nodeID, addr)
}
