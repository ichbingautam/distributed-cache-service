package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	pb "distributed-cache-service/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. Build and Start Server
	log.Println("Starting server...")
	cmd := exec.Command("./server", "-node_id", "test_node", "-raft_addr", ":12000", "-http_addr", ":8090", "-grpc_addr", ":50055", "-bootstrap", "-raft_dir", "raft_verify_test")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = os.RemoveAll("raft_verify_test")
	}()

	// Wait for startup
	time.Sleep(3 * time.Second)

	// 2. HTTP Verification (Backward Compatibility)
	log.Println("Testing HTTP API (Backward Compatibility)...")
	httpSetURL := "http://localhost:8090/set?key=http_key&value=http_val"
	if err := httpGet(httpSetURL); err != nil {
		log.Fatalf("HTTP Set failed: %v", err)
	}

	// Allow Raft to commit
	time.Sleep(1 * time.Second)

	httpGetURL := "http://localhost:8090/get?key=http_key"
	val, err := httpGetBody(httpGetURL)
	if err != nil {
		log.Fatalf("HTTP Get failed: %v", err)
	}
	if val != "http_val" {
		log.Fatalf("HTTP Get mismatch: expected 'http_val', got '%s'", val)
	}
	log.Println("✅ HTTP API Verified")

	// 3. gRPC Verification (New Feature)
	log.Println("Testing gRPC API (New Feature)...")
	conn, err := grpc.NewClient("localhost:50055", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC: %v", err)
	}
	defer conn.Close()

	client := pb.NewCacheServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// gRPC Set
	_, err = client.Set(ctx, &pb.SetRequest{Key: "grpc_key", Value: "grpc_val", Ttl: 60})
	if err != nil {
		log.Fatalf("gRPC Set failed: %v", err)
	}

	// Allow Raft to commit
	time.Sleep(1 * time.Second)

	// gRPC Get
	resp, err := client.Get(ctx, &pb.GetRequest{Key: "grpc_key"})
	if err != nil {
		log.Fatalf("gRPC Get failed: %v", err)
	}
	if !resp.Found || resp.Value != "grpc_val" {
		log.Fatalf("gRPC Get mismatch: expected 'grpc_val', got '%s'", resp.Value)
	}
	log.Println("✅ gRPC API Verified")

	// 4. Cross-Protocol Verification
	// Can HTTP read gRPC data?
	log.Println("Testing Cross-Protocol Access...")
	val, err = httpGetBody("http://localhost:8090/get?key=grpc_key")
	if err != nil {
		log.Fatalf("HTTP read of gRPC data failed: %v", err)
	}
	if val != "grpc_val" {
		log.Fatalf("Cross-protocol mismatch: expected 'grpc_val', got '%s'", val)
	}
	log.Println("✅ Cross-Protocol Verified")
}

func httpGet(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}
	return nil
}

func httpGetBody(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
