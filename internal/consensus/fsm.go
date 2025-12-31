package consensus

import (
	"encoding/json"
	"fmt"
	"io"

	"distributed-cache-service/internal/core/service"
	"distributed-cache-service/internal/store"

	"github.com/hashicorp/raft"
)

// FSM implements raft.FSM interface
// FSM (Finite State Machine) implements the raft.FSM interface.
// It is responsible for applying committed log entries to the underlying key-value store
// and managing snapshots of the state.
type FSM struct {
	store *store.Store
}

// NewFSM creates a new FSM instance backed by the provided store.
func NewFSM(s *store.Store) *FSM {
	return &FSM{
		store: s,
	}
}

// Apply applies a committed Raft log entry to the key-value store.
// It unmarshals the command (Set/Delete) and executes it against the backend store.
// This method is invoked by the Raft leader after consensus is reached.
func (f *FSM) Apply(log *raft.Log) interface{} {
	var c service.Command
	if err := json.Unmarshal(log.Data, &c); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch c.Op {
	case service.SetOp:
		f.store.Set(c.Key, c.Value, c.TTL)
	case service.DeleteOp:
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
		_ = sink.Cancel()
		return err
	}

	return sink.Close()
}

func (s *Snapshot) Release() {
	// No-op
}
