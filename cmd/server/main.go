package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"distributed-cache-service/internal/consensus"
	"distributed-cache-service/internal/core/service"
	"distributed-cache-service/internal/store"

	_ "net/http/pprof" // Register pprof handlers

	"github.com/hashicorp/raft"
)

func main() {
	var (
		nodeID    = flag.String("node_id", "node1", "Node ID")
		httpAddr  = flag.String("http_addr", ":8080", "HTTP Server address")
		raftAddr  = flag.String("raft_addr", ":11000", "Raft communication address")
		raftDir   = flag.String("raft_dir", "raft_data", "Raft data directory")
		bootstrap = flag.Bool("bootstrap", false, "Bootstrap the cluster (only for the first node)")
		joinAddr  = flag.String("join", "", "Address of the leader to join")
	)
	flag.Parse()

	// Ensure raft storage directory exists
	os.MkdirAll(*raftDir, 0700)

	// Initialize Store and FSM
	kvStore := store.New()
	fsm := consensus.NewFSM(kvStore)

	// Setup Raft
	raftSys, err := consensus.SetupRaft(*raftDir, *nodeID, *raftAddr, fsm)
	if err != nil {
		log.Fatalf("Failed to setup Raft: %v", err)
	}

	// Create consensus adapter and service
	raftNode := &consensus.RaftNode{Raft: raftSys}
	svc := service.New(kvStore, raftNode)

	// Bootstrap if requested
	if *bootstrap {
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(*nodeID),
					Address: raft.ServerAddress(*raftAddr),
				},
			},
		}
		f := raftSys.BootstrapCluster(cfg)
		if err := f.Error(); err != nil {
			log.Printf("Failed to bootstrap cluster: %v", err)
		}
	} else if *joinAddr != "" {
		// Try to join an existing cluster
		if err := joinCluster(*nodeID, *raftAddr, *joinAddr); err != nil {
			log.Fatalf("Failed to join cluster: %v", err)
		}
	}

	// HTTP handlers
	http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		val := r.URL.Query().Get("value")
		if key == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}

		err := svc.Set(r.Context(), key, val, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("ok"))
	})

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}

		val, err := svc.Get(r.Context(), key)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write([]byte(val))
	})

	http.HandleFunc("/join", func(w http.ResponseWriter, r *http.Request) {
		nodeID := r.URL.Query().Get("node_id")
		remoteAddr := r.URL.Query().Get("addr")

		if nodeID == "" || remoteAddr == "" {
			http.Error(w, "missing node_id or addr", http.StatusBadRequest)
			return
		}

		if err := svc.Join(r.Context(), nodeID, remoteAddr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("joined"))
	})

	log.Printf("Server listening on %s (Raft: %s)...", *httpAddr, *raftAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

func joinCluster(nodeID, raftAddr, joinAddr string) error {
	url := fmt.Sprintf("http://%s/join?node_id=%s&addr=%s", joinAddr, nodeID, raftAddr)
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to join: %s", resp.Status)
	}
	return nil
}
