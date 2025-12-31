package consensus

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	// Added for string containment check

	// Added for string containment check

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// BufferedConn wraps a net.Conn to replay a peeked byte
type BufferedConn struct {
	net.Conn
	peeked []byte
}

func (c *BufferedConn) Read(p []byte) (n int, err error) {
	if len(c.peeked) > 0 {
		n = copy(p, c.peeked)
		c.peeked = c.peeked[n:]
		if len(c.peeked) == 0 {
			c.peeked = nil
		}
		return n, nil
	}
	return c.Conn.Read(p)
}

// RaftListener intercepts HTTP connections (health checks) and filters them out
type RaftListener struct {
	net.Listener
}

func (l *RaftListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		// Set a short read deadline to peek at the first byte
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		peek := make([]byte, 1)
		n, err := conn.Read(peek)
		conn.SetReadDeadline(time.Time{}) // Reset deadline

		if err != nil {
			conn.Close()
			continue // Handshake failed, ignore
		}

		if n == 0 {
			conn.Close()
			continue
		}

		// Check if it looks like HTTP (G=GET, H=HEAD, P=POST, C=CONNECT, O=OPTIONS)
		// Raft RPC types are binary, unlikely to match these ASCII upper-case letters first.
		// Valid Raft RPC types (as of hashicorp/raft):
		// 0=RPCInstallSnapshot, 1=RPCAppendEntries, 2=RPCRequestVote, 3=RPCTimeoutNow
		// So checking for ASCII chars is safe.
		b := peek[0]
		if b == 'G' || b == 'H' || b == 'P' || b == 'C' || b == 'O' || b == 'D' {
			// It is likely HTTP. Respond with 200 OK
			conn.Write([]byte("HTTP/1.1 200 OK\r\nConnection: close\r\nContent-Length: 2\r\n\r\nok"))
			conn.Close()
			continue // Drop this connection, don't return to Raft
		}

		// It is not HTTP, pass it to Raft with the peeked byte
		return &BufferedConn{Conn: conn, peeked: peek}, nil
	}
}

func (l *RaftListener) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", string(address), timeout)
}

// SetupRaft initializes and starts a Raft node.
func SetupRaft(dir, nodeId, bindAddr, advertiseAddr string, fsm *FSM) (*raft.Raft, error) {
	// Setup Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeId)
	// config.Logger = hclog.New(&hclog.LoggerOptions{Output: os.Stderr, Level: hclog.Error, Name: "raft"})

	// Create a custom listener that traps HTTP health checks
	realListener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return nil, err
	}
	raftListener := &RaftListener{Listener: realListener}

	transport := raft.NewNetworkTransport(raftListener, 3, 10*time.Second, os.Stderr)

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
