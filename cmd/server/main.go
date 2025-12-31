package main

import (
	"flag"
	"fmt"
	"log"
	"net"
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
		nodeID   = flag.String("node_id", "node1", "Node ID")
		httpAddr = flag.String("http_addr", ":8080", "HTTP Server address")

		raftAddr  = flag.String("raft_addr", ":11000", "Raft communication address")
		raftAdv   = flag.String("raft_advertise", "", "Advertised Raft address (defaults to local IP if raft_addr is generic)")
		raftDir   = flag.String("raft_dir", "raft_data", "Raft data directory")
		bootstrap = flag.Bool("bootstrap", false, "Bootstrap the cluster (only for the first node)")
		joinAddr  = flag.String("join", "", "Address of the leader to join")
	)
	flag.Parse()

	// Check environment variable for PORT (e.g., Render)
	if port := os.Getenv("PORT"); port != "" {
		*httpAddr = ":" + port
	}

	// Ensure raft storage directory exists
	os.MkdirAll(*raftDir, 0700)

	// Initialize Store and FSM
	kvStore := store.New()
	fsm := consensus.NewFSM(kvStore)

	// Determine advertise address
	// Determine advertise address and bind address
	var bindAddr string
	advertiseAddr := *raftAdv

	host, port, err := net.SplitHostPort(*raftAddr)
	if err != nil {
		log.Fatalf("Invalid raft_addr: %v", err)
	}

	if host == "" || host == "0.0.0.0" {
		// Resolve local IP
		addr, err := getLocalIP()
		if err != nil {
			log.Fatalf("Could not determine local IP: %v", err)
		}
		// Bind to the specific local IP to avoid unwanted traffic on 0.0.0.0 from LB health checks
		bindAddr = fmt.Sprintf("%s:%s", addr, port)
		if advertiseAddr == "" {
			advertiseAddr = bindAddr
		}
	} else {
		bindAddr = *raftAddr
		if advertiseAddr == "" {
			advertiseAddr = *raftAddr
		}
	}

	// Setup Raft
	raftSys, err := consensus.SetupRaft(*raftDir, *nodeID, bindAddr, advertiseAddr, fsm)
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

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no private IP address found")
}
