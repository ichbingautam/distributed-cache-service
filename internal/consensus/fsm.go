package consensus

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"distributed-cache-service/internal/store"

	"github.com/hashicorp/raft"
)

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

// FSM implements raft.FSM interface
type FSM struct {
	store *store.Store
	mu    sync.Mutex
}

func NewFSM(s *store.Store) *FSM {
	return &FSM{
		store: s,
	}
}

// Apply applies a Raft log entry to the key-value store.
func (f *FSM) Apply(log *raft.Log) interface{} {
	var c Command
	if err := json.Unmarshal(log.Data, &c); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch c.Op {
	case SetOp:
		f.store.Set(c.Key, c.Value, c.TTL)
	case DeleteOp:
		f.store.Delete(c.Key)
	default:
		return fmt.Errorf("unknown command op: %s", c.Op)
	}
	return nil
}

// Snapshot returns a snapshot object
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	// In a real system, we might want to copy the map efficiently.
	// For now, we rely on the store's Snapshot method which locks the store.
	return &Snapshot{store: f.store}, nil
}

// Restore restores the key-value store from a snapshot.
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	return f.store.Restore(rc)
}

// Snapshot implementation
type Snapshot struct {
	store *store.Store
}

func (s *Snapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode data from the store into the sink
		if err := s.store.Snapshot(sink); err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		sink.Cancel()
		return err
	}

	return sink.Close()
}

func (s *Snapshot) Release() {
	// No-op
}
