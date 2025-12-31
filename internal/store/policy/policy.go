package policy

// EvictionPolicy defines the interface for eviction algorithms.
// Implementations allow the store to decouple capacity management from storage logic.
type EvictionPolicy interface {
	// OnAccess is called when a key is accessed (read or updated).
	// This allows policies (like LRU/LFU) to update their internal state.
	OnAccess(key string)

	// OnAdd is called when a new key is added to the store.
	OnAdd(key string)

	// OnRemove is called when a key is manually deleted from the store.
	OnRemove(key string)

	// SelectVictim returns the key that should be evicted according to the policy.
	// Returns an empty string if no victim is available (e.g., empty store).
	SelectVictim() string
}
