package consensus

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// SetupRaft initializes and starts a Raft node.
func SetupRaft(dir, nodeId, bindAddr string, fsm *FSM) (*raft.Raft, error) {
	// Setup Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeId)
	// config.Logger = hclog.New(&hclog.LoggerOptions{Output: os.Stderr, Level: hclog.Error, Name: "raft"})

	// Setup Raft communication
	addr, err := net.ResolveTCPAddr("tcp", bindAddr)
	if err != nil {
		return nil, err
	}

	transport, err := raft.NewTCPTransport(bindAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Create the snapshot store. This allows the Raft to truncate the log.
	snapshotStore, err := raft.NewFileSnapshotStore(dir, 2, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Create the log store and stable store
	var logStore raft.LogStore
	var stableStore raft.StableStore

	boltDir := filepath.Join(dir, "raft.db")
	boltDB, err := raftboltdb.NewBoltStore(boltDir)
	if err != nil {
		return nil, fmt.Errorf("new bolt store: %w", err)
	}
	logStore = boltDB
	stableStore = boltDB

	// Instantiate the Raft systems
	ra, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("new raft: %w", err)
	}

	return ra, nil
}

// Wrapper to satisfy ports.Consensus interface
type RaftNode struct {
	Raft *raft.Raft
}

func (n *RaftNode) Apply(cmd []byte) error {
	f := n.Raft.Apply(cmd, 500*time.Millisecond) // Lower timeout
	return f.Error()
}

func (n *RaftNode) AddVoter(id, addr string) error {
	f := n.Raft.AddVoter(raft.ServerID(id), raft.ServerAddress(addr), 0, 0)
	return f.Error()
}

func (n *RaftNode) IsLeader() bool {
	return n.Raft.State() == raft.Leader
}
