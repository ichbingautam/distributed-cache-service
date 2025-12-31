package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings" // Added for strings.ToLower
	"time"

	"distributed-cache-service/internal/consensus"
	"distributed-cache-service/internal/core/service"
	"distributed-cache-service/internal/sharding"
	"distributed-cache-service/internal/store"
	"distributed-cache-service/internal/store/policy" // Added for eviction policies

	_ "net/http/pprof" // Register pprof handlers

	"github.com/hashicorp/raft"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	// Added for raft-boltdb
	grpcAdapter "distributed-cache-service/internal/grpc"
	pb "distributed-cache-service/proto"
)

func main() {
	// ... existing flags ...
	var (
		nodeID       = flag.String("node_id", "node1", "Node ID")
		httpAddr     = flag.String("http_addr", ":8080", "HTTP Server address")
		raftAddr     = flag.String("raft_addr", ":11000", "Raft communication address")
		raftAdv      = flag.String("raft_advertise", "", "Advertised Raft address (defaults to local IP if raft_addr is generic)")
		raftDir      = flag.String("raft_dir", "raft_data", "Raft data directory")
		bootstrap    = flag.Bool("bootstrap", false, "Bootstrap the cluster (only for the first node)")
		joinAddr     = flag.String("join", "", "Address of the leader to join")
		maxItems     = flag.Int("max_items", 0, "Maximum number of items in the cache (0 = unlimited)")
		evictionPol  = flag.String("eviction_policy", "lru", "Eviction policy: lru, fifo, lfu, random, none")
		grpcAddr     = flag.String("grpc_addr", ":50051", "gRPC Server address")
		virtualNodes = flag.Int("virtual_nodes", 100, "Number of virtual nodes for consistent hashing")
		consistency  = flag.String("consistency", "strong", "Consistency mode: strong, eventual")
	)
	// -------------------------------------------------------------------------
	// 1. Parsing Configuration
	// -------------------------------------------------------------------------
	flag.Parse()

	// Check environment variable for PORT (e.g., Render)
	if port := os.Getenv("PORT"); port != "" {
		*httpAddr = ":" + port
	}

	if err := os.MkdirAll(*raftDir, 0700); err != nil {
		log.Fatalf("Failed to create raft directory: %v", err)
	}

	// Configure Store with options
	var storeOpts []store.Option
	if *maxItems > 0 {
		storeOpts = append(storeOpts, store.WithCapacity(*maxItems))
		var p policy.EvictionPolicy
		switch strings.ToLower(*evictionPol) {
		case "lru":
			p = policy.NewLRU()
		case "fifo":
			p = policy.NewFIFO()
		case "lfu":
			p = policy.NewLFU()
		case "random":
			p = policy.NewRandom()
		case "none":
			p = nil
		default:
			log.Printf("Unknown eviction policy '%s', defaulting to LRU", *evictionPol)
			p = policy.NewLRU()
		}
		if p != nil {
			storeOpts = append(storeOpts, store.WithPolicy(p))
		}
	}

	// -------------------------------------------------------------------------
	// 2. Core Domain & Storage Setup
	// -------------------------------------------------------------------------
	// Initialize Sharding Ring (Virtual Nodes)
	// Note: Currently local-only view, but prepared for Smart Client / Partitioning
	_ = sharding.New(*virtualNodes, nil)

	// Initialize Store and FSM
	kvStore := store.New(storeOpts...)
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

	// -------------------------------------------------------------------------
	// 3. Raft Consensus Setup
	// -------------------------------------------------------------------------
	// Setup Raft
	raftSys, err := consensus.SetupRaft(*raftDir, *nodeID, bindAddr, advertiseAddr, fsm)
	if err != nil {
		log.Fatalf("Failed to setup Raft: %v", err)
	}

	// Validate Consistency Mode
	var consistencyMode service.ConsistencyMode
	switch strings.ToLower(*consistency) {
	case "strong":
		consistencyMode = service.ConsistencyStrong
	case "eventual":
		consistencyMode = service.ConsistencyEventual
	default:
		log.Printf("Unknown consistency mode '%s', defaulting to strong", *consistency)
		consistencyMode = service.ConsistencyStrong
	}

	// Create consensus adapter and service
	raftNode := &consensus.RaftNode{Raft: raftSys}
	svc := service.New(kvStore, raftNode, consistencyMode)

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

	// -------------------------------------------------------------------------
	// 4. HTTP API & Server Start
	// -------------------------------------------------------------------------
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

		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
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
		if _, err := w.Write([]byte(val)); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
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
		if _, err := w.Write([]byte("joined")); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
	})

	// Health Check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
	})

	// Prometheus Metrics
	http.Handle("/metrics", promhttp.Handler())

	// -------------------------------------------------------------------------
	// 5. gRPC Server Start
	// -------------------------------------------------------------------------
	// Assuming I fix flag definition separately.
	go func() {
		lis, err := net.Listen("tcp", *grpcAddr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterCacheServiceServer(grpcServer, grpcAdapter.New(svc))
		log.Printf("gRPC server listening on %s", *grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	log.Printf("Server listening on %s (Raft: %s)...", *httpAddr, *raftAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

// joinCluster sends a request to an existing node to add this node to the cluster.
// It hits the /join endpoint of the target leader.
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

// getLocalIP returns the first non-loopback private IP address of the machine.
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
