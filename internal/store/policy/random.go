package policy

import (
	"math/rand"
	"sync"
	"time"
)

// RandomPolicy implements a random eviction strategy.
type RandomPolicy struct {
	mu    sync.Mutex
	items []string
	rnd   *rand.Rand
}

// NewRandom creates a new Random policy instance with a time-based seed.
func NewRandom() *RandomPolicy {
	src := rand.NewSource(time.Now().UnixNano())
	return &RandomPolicy{
		items: make([]string, 0),
		rnd:   rand.New(src),
	}
}

// newRandomWithRand creates a new Random policy with a specific random source (for testing).
func newRandomWithRand(r *rand.Rand) *RandomPolicy {
	return &RandomPolicy{
		items: make([]string, 0),
		rnd:   r,
	}
}

// OnAccess acts as a no-op for the Random policy.
// Access patterns do not influence eviction probability in this strategy.
func (p *RandomPolicy) OnAccess(key string) {
	// No-op for Random
}

// OnAdd adds a new key to the candidate pool.
// It acquires a lock to ensure thread safety.
func (p *RandomPolicy) OnAdd(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = append(p.items, key)
}

// OnRemove removes a key from the candidate pool.
// It performs an O(N) search to find the key, then performs a swap-remove
// to delete it efficiently without preserving order.
func (p *RandomPolicy) OnRemove(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// O(N) removal finding the key
	for i, k := range p.items {
		if k == key {
			// Swap with last element and truncate
			lastIdx := len(p.items) - 1
			p.items[i] = p.items[lastIdx]
			p.items = p.items[:lastIdx]
			return
		}
	}
}

// Evict selects a victim to be removed from the cache.
// It delegates to SelectVictim.
func (p *RandomPolicy) Evict() string {
	return p.SelectVictim()
}

// SelectVictim chooses a random key from the current items.
// It uses a uniform random distribution provided by the local source.
// Returns an empty string if the policy has no items.
func (p *RandomPolicy) SelectVictim() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.items) == 0 {
		return ""
	}
	idx := p.rnd.Intn(len(p.items))
	return p.items[idx]
}
