package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "distributed-cache-service/proto"
)

type mockService struct {
	getFunc    func(ctx context.Context, key string) (string, error)
	setFunc    func(ctx context.Context, key, value string, ttl time.Duration) error
	deleteFunc func(ctx context.Context, key string) error
	joinFunc   func(ctx context.Context, id, addr string) error
}

func (m *mockService) Get(ctx context.Context, key string) (string, error) {
	return m.getFunc(ctx, key)
}
func (m *mockService) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return m.setFunc(ctx, key, value, ttl)
}
func (m *mockService) Delete(ctx context.Context, key string) error {
	return m.deleteFunc(ctx, key)
}
func (m *mockService) Join(ctx context.Context, id, addr string) error {
	return m.joinFunc(ctx, id, addr)
}

func TestAdapter_Get(t *testing.T) {
	mock := &mockService{
		getFunc: func(ctx context.Context, key string) (string, error) {
			if key == "found" {
				return "value", nil
			}
			return "", errors.New("not found")
		},
	}
	adapter := New(mock)

	// Test Found
	resp, err := adapter.Get(context.Background(), &pb.GetRequest{Key: "found"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Found || resp.Value != "value" {
		t.Errorf("expected found=true value='value', got found=%v value='%s'", resp.Found, resp.Value)
	}

	// Test Not Found
	resp, err = adapter.Get(context.Background(), &pb.GetRequest{Key: "missing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Found {
		t.Errorf("expected found=false")
	}
}
