package policy

import (
	"math/rand"
	"sync"
	"time"
)

// RandomPolicy implements a random eviction strategy.
type RandomPolicy struct {
	mu   sync.Mutex
	keys []string // We maintain a slice for true implementation, or iterate map.
	// Map iteration in Go is random.
	// If we use map iteration, it is O(1) amortized but requires 'range' which might be weird to break.
	// Actually keeping a slice allows O(1) Pick but O(N) Remove.
	// Iterating map is O(1) for finding *one* key (statistically).
	// Let's use Slice + Map to support O(1) Remove (Swap Remove).
	keyMap map[string]int // Maps key to index in slice
}

// NewRandom creates a new Random policy instance.
func NewRandom() *RandomPolicy {
	rand.Seed(time.Now().UnixNano())
	return &RandomPolicy{
		keys:   make([]string, 0),
		keyMap: make(map[string]int),
	}
}

func (p *RandomPolicy) OnAccess(key string) {
	// No-op
}

func (p *RandomPolicy) OnAdd(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.keyMap[key]; ok {
		return
	}
	p.keys = append(p.keys, key)
	p.keyMap[key] = len(p.keys) - 1
}

func (p *RandomPolicy) OnRemove(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	idx, ok := p.keyMap[key]
	if !ok {
		return
	}
	// Swap remove
	lastIdx := len(p.keys) - 1
	lastKey := p.keys[lastIdx]

	p.keys[idx] = lastKey
	p.keyMap[lastKey] = idx

	p.keys = p.keys[:lastIdx]
	delete(p.keyMap, key)
}

func (p *RandomPolicy) SelectVictim() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.keys) == 0 {
		return ""
	}
	idx := rand.Intn(len(p.keys))
	return p.keys[idx]
}
