package store

import (
	"testing"
	"time"
)

func TestStore_SetGet(t *testing.T) {
	s := New()
	key := "test_key"
	val := "test_val"

	s.Set(key, val, 0)

	got, found := s.Get(key)
	if !found {
		t.Fatalf("expected key %s to be found", key)
	}
	if got != val {
		t.Errorf("expected value %s, got %s", val, got)
	}
}

func TestStore_TTL(t *testing.T) {
	s := New()
	key := "ttl_key"
	val := "ttl_val"

	// Set with short TTL
	s.Set(key, val, 100*time.Millisecond)

	// Should be found immediately
	_, found := s.Get(key)
	if !found {
		t.Fatal("key should be found immediately")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should not be found after expiration
	_, found = s.Get(key)
	if found {
		t.Fatal("key should have expired")
	}
}

func TestStore_Delete(t *testing.T) {
	s := New()
	s.Set("key", "val", 0)
	s.Delete("key")
	_, found := s.Get("key")
	if found {
		t.Fatal("key should have been deleted")
	}
}
