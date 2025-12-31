package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock implementation of ports.Storage
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Get(key string) (string, bool) {
	args := m.Called(key)
	return args.String(0), args.Bool(1)
}

func (m *MockStorage) Set(key, value string, ttl time.Duration) {
	m.Called(key, value, ttl)
}

func (m *MockStorage) Delete(key string) {
	m.Called(key)
}

// MockConsensus is a mock implementation of ports.Consensus
type MockConsensus struct {
	mock.Mock
}

func (m *MockConsensus) Apply(cmd []byte) error {
	args := m.Called(cmd)
	return args.Error(0)
}

func (m *MockConsensus) AddVoter(id, addr string) error {
	args := m.Called(id, addr)
	return args.Error(0)
}

func (m *MockConsensus) IsLeader() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestServiceImpl_Get(t *testing.T) {
	mockStore := new(MockStorage)
	mockConsensus := new(MockConsensus)
	svc := New(mockStore, mockConsensus)

	ctx := context.Background()

	// Test Found
	mockStore.On("Get", "key1").Return("value1", true)
	val, err := svc.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	// Test Not Found
	mockStore.On("Get", "unknown").Return("", false)
	val, err = svc.Get(ctx, "unknown")
	assert.Error(t, err)
	assert.Empty(t, val)
}

func TestServiceImpl_Set(t *testing.T) {
	mockStore := new(MockStorage)
	mockConsensus := new(MockConsensus)
	svc := New(mockStore, mockConsensus)

	ctx := context.Background()
	key := "key1"
	val := "value1"
	ttl := time.Minute

	// Service layer should wrap command and call Consensus.Apply
	// We don't check exact JSON bytes to avoid fragility, but we check if Apply is called
	mockConsensus.On("Apply", mock.Anything).Return(nil)

	err := svc.Set(ctx, key, val, ttl)
	assert.NoError(t, err)
	mockConsensus.AssertExpectations(t)
}

func TestServiceImpl_Delete(t *testing.T) {
	mockStore := new(MockStorage)
	mockConsensus := new(MockConsensus)
	svc := New(mockStore, mockConsensus)

	ctx := context.Background()
	mockConsensus.On("Apply", mock.Anything).Return(nil)

	err := svc.Delete(ctx, "key1")
	assert.NoError(t, err)
	mockConsensus.AssertExpectations(t)
}

func TestServiceImpl_Join(t *testing.T) {
	mockStore := new(MockStorage)
	mockConsensus := new(MockConsensus)
	svc := New(mockStore, mockConsensus)

	ctx := context.Background()
	mockConsensus.On("AddVoter", "node2", "addr2").Return(nil)

	err := svc.Join(ctx, "node2", "addr2")
	assert.NoError(t, err)
	mockConsensus.AssertExpectations(t)
}
