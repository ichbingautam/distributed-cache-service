package grpc

import (
	"context"
	"time"

	"distributed-cache-service/internal/core/ports"
	pb "distributed-cache-service/proto"
)

// Adapter implements the generated CacheServiceServer interface.
type Adapter struct {
	pb.UnimplementedCacheServiceServer
	service ports.CacheService
}

// New creates a new gRPC adapter.
func New(service ports.CacheService) *Adapter {
	return &Adapter{service: service}
}

// Get retrieves a value from the cache.
func (s *Adapter) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	val, err := s.service.Get(ctx, req.Key)
	if err != nil {
		// Verify if it's a not found error or other error
		// For simplicity, we assume error means not found for now, or we can check string
		return &pb.GetResponse{Value: "", Found: false}, nil
	}
	return &pb.GetResponse{Value: val, Found: true}, nil
}

// Set stores a value in the cache.
func (s *Adapter) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	err := s.service.Set(ctx, req.Key, req.Value, time.Duration(req.Ttl)*time.Second)
	if err != nil {
		return &pb.SetResponse{Success: false}, err
	}
	return &pb.SetResponse{Success: true}, nil
}

// Delete removes a value from the cache.
func (s *Adapter) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := s.service.Delete(ctx, req.Key)
	if err != nil {
		return &pb.DeleteResponse{Success: false}, err
	}
	return &pb.DeleteResponse{Success: true}, nil
}
