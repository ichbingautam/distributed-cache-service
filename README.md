# Distributed Cache Service

High-performance distributed cache in Go using Raft for consensus and Consistent Hashing for sharding.

## Architecture Patterns

- **Hexagonal Architecture (Ports & Adapters)**:
  - **Core**: Business logic in `internal/core/service`.
  - **Ports**: Interfaces defined in `internal/core/ports`.
  - **Adapters**: `consensus` (Raft), `store` (Memory), and `cmd/server` (HTTP) act as adapters.
- **Sharding**: Consistent Hashing (Ring) to minimize keys remap.
- **Consensus**: Raft (Leader-Follower) for strong consistency within a shard.
- **Storage**: In-Memory Thread-Safe Map with TTL support.
- **API**: HTTP (Fallback) / gRPC (Planned).

## Project Structure

```
├── cmd
│   └── server          # Main entry point for the application
├── internal
│   ├── consensus       # Raft implementation and FSM adapter
│   ├── core
│       ├── ports       # Interfaces for Service, Storage, and Consensus
│       └── service     # Business logic and Command definitions
│   ├── sharding        # Consistent Hashing implementation
│   └── store           # In-Memory key-value store implementation
├── k8s                 # Kubernetes manifests (StatefulSet, Service)
├── proto               # Protobuf definitions (for future gRPC)
└── raft_data           # Directory for Raft logs and snapshots (created at runtime)
```

## Configuration

The server accepts the following command-line flags:

| Flag          | Default      | Description                                      |
|---------------|--------------|--------------------------------------------------|
| `-node_id`    | `node1`      | Unique identifier for the Raft node.             |
| `-http_addr`  | `:8080`      | Address to bind the HTTP server.                 |
| `-raft_addr`  | `:11000`     | Address to bind the Raft transport.              |
| `-raft_dir`   | `raft_data`  | Directory to store Raft data (logs/snapshots).   |
| `-bootstrap`  | `false`      | Set to `true` to bootstrap a new cluster (leader).|
| `-join`       | `""`         | Address of an existing leader to join.           |

## Deployment

### Kubernetes (Production)

1. **Build the Docker Image**:
   ```bash
   docker build -t distributed-cache-service:latest .
   ```

2. **Deploy to Cluster**:
   ```bash
   kubectl apply -f k8s/
   ```

3. **Verify Deployment**:
   ```bash
   kubectl get pods -l app=cache-service
   kubectl logs cache-node-0
   ```

### Local Development

To run a 3-node cluster locally:

**Node 1 (Leader):**
```bash
./server -node_id node1 -http_addr :8081 -raft_addr :11001 -raft_dir raft_node1 -bootstrap
```

**Node 2 (Follower):**
```bash
./server -node_id node2 -http_addr :8082 -raft_addr :11002 -raft_dir raft_node2 -join localhost:8081
```

**Node 3 (Follower):**
```bash
./server -node_id node3 -http_addr :8083 -raft_addr :11003 -raft_dir raft_node3 -join localhost:8081
```

## API Documentation

### 1. Set Key
Sets a value for a key. This operation is replicated via Raft.

- **Endpoint**: `GET /set` (for demo convenience, typically POST)
- **Parameters**:
  - `key`: The key to set.
  - `value`: The value to store.
  - `ttl`: (Optional) Time to live in seconds.
- **Response**: `ok` or error message.

### 2. Get Key
Retrieves a value. This operation is strongly consistent (read from Leader for simple implementation).

- **Endpoint**: `GET /get`
- **Parameters**:
  - `key`: The key to retrieve.
- **Response**: The value string or `not found`.

### 3. Join Cluster
Adds a new node to the Raft cluster.

- **Endpoint**: `GET /join`
- **Parameters**:
  - `node_id`: Unique ID of the new node.
  - `addr`: Raft address of the new node (e.g., `127.0.0.1:11000`).
- **Response**: `joined` or error message.

## Usage Examples

**Start the Server:**
```bash
./server -node_id node1 -http_addr :8080 -raft_addr :11000 -raft_dir raft_node1 -bootstrap
```

**Write a Value:**
```bash
curl "http://localhost:8080/set?key=hello&value=world"
```

**Read a Value:**
```bash
curl "http://localhost:8080/get?key=hello"
```

## Sequence Diagram: Write Operation

```mermaid
sequenceDiagram
    participant Client
    participant Leader
    participant Follower1
    participant Follower2
    participant FSM
    participant Store

    Client->>Leader: HTTP /set?key=A&value=1
    Leader->>Leader: Create Log Entry
    Leader->>Follower1: AppendEntries(A=1)
    Leader->>Follower2: AppendEntries(A=1)
    Follower1-->>Leader: Acknowledge
    Follower2-->>Leader: Acknowledge
    Leader->>Leader: Commit Log
    Leader->>FSM: Apply(A=1)
    FSM->>Store: Set("A", "1")
    Leader-->>Client: 200 OK "ok"
```

## Running Tests

### Unit Tests
```bash
go test ./internal/...
```

### Performance Benchmark
```bash
go test -bench=. ./internal/store
```

## Profiling

The service exposes `pprof` endpoints at `/debug/pprof/`.

### 1. CPU Profile
```bash
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

### 2. Heap (Memory) Profile
```bash
go tool pprof http://localhost:8080/debug/pprof/heap
```

### 3. Goroutine Blocking Profile
```bash
go tool pprof http://localhost:8080/debug/pprof/block
```

## Path to 10M RPS (Scaling Strategy)

Achieving 10 Million Requests Per Second requires evolving this MVP with the following architectural optimizations:

1.  **Multi-Raft / Sharded Consensus**:
    - **Current**: Single Raft group for the whole cluster.
    - **Bottleneck**: Leader becomes the write bottleneck.
    - **Solution**: Split data into partitions (Ranges/Shards). Each partition has its own Raft Consensus Group. This allows writes to scale linearly with the number of nodes (like CockroachDB or TiKV).

2.  **Smart Client (Client-Side Routing)**:
    - **Current**: Client talks to any node (or one via DNS).
    - **Solution**: Clients should fetch the **Hash Ring / Partition Map** and route requests directly to the correct node (or Leader for writes), eliminating an extra network hop.

3.  **Transport Layer Optimization**:
    - **Current**: HTTP/JSON.
    - **Solution**: Switch to **gRPC with Protobuf** for smaller payload size and connection multiplexing. For extreme low latency, consider custom binary protocols over TCP/UDP.

4.  **Hardware & OS Tuning**:
    - Use High-Performance Networking (SR-IOV, DPDK) if CPU bound.
    - Kernel tuning: `SO_REUSEPORT`, increased file descriptors, TCP Fast Open.

5.  **Hot Key Handling**:
    - Implement **Local Caching** (Near Cache) on the client side for extremely hot keys.
    - Use "Shadow Replicas" to serve reads for hot keys from more nodes than just the consensus group.
