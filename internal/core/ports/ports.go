package ports

import (
	"context"
	"time"
)

// CacheService maps incoming requests to business logic methods.
// It defines the primary use cases for the distributed cache system.
type CacheService interface {
	// Get retrieves a value for a given key.
	Get(ctx context.Context, key string) (string, error)
	// Set stores a value for a given key with an optional TTL.
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Delete removes a key from the cache.
	Delete(ctx context.Context, key string) error
	// Join adds a new node to the distributed cluster.
	Join(ctx context.Context, nodeID, addr string) error
}

// Storage defines the interface for underlying data persistence/storage.
// Implementations should be thread-safe.
type Storage interface {
	// Get retrieves the value and existence boolean for a key.
	Get(key string) (string, bool)
	// Set stores the key-value pair with an expiration duration.
	Set(key, value string, ttl time.Duration)
	// Delete removes the key from storage.
	Delete(key string)
}

// Consensus defines the interface for distributed agreement/replication.
type Consensus interface {
	// Apply replicates a state-changing command to the cluster.
	Apply(cmd []byte) error
	// AddVoter adds a new voting member to the cluster.
	AddVoter(id, addr string) error
	// IsLeader checks if the current node is the cluster leader.
	IsLeader() bool
	// VerifyLeader checks if the current node is the leader and can serve consistent reads.
	VerifyLeader() error
}
