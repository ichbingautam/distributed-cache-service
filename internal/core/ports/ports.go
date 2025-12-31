package ports

import (
	"context"
	"time"
)

// CacheService maps incoming requests to business logic
type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Join(ctx context.Context, nodeID, addr string) error
}

// Storage defines the interface for data persistence/storage
type Storage interface {
	Get(key string) (string, bool)
	Set(key, value string, ttl time.Duration)
	Delete(key string)
}

// Consensus defines the interface for distributed agreement
type Consensus interface {
	Apply(cmd []byte) error
	AddVoter(id, addr string) error
	IsLeader() bool
}
