package policy

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLRUPolicy(t *testing.T) {
	lru := NewLRU()

	// Add A, B, C
	lru.OnAdd("A")
	lru.OnAdd("B")
	lru.OnAdd("C")

	// Order should be C, B, A (Most Recent -> Least Recent)
	// Access A
	lru.OnAccess("A")
	// Order: A, C, B. Victim: B.

	assert.Equal(t, "B", lru.SelectVictim())

	// Remove B logic
	lru.OnRemove("B")

	// Now victim is C
	assert.Equal(t, "C", lru.SelectVictim())
}

func TestFIFOPolicy(t *testing.T) {
	fifo := NewFIFO()

	// Add A, B, C
	fifo.OnAdd("A")
	fifo.OnAdd("B")
	fifo.OnAdd("C")

	// Access A (should not allow A to escape eviction)
	fifo.OnAccess("A")

	// Victim should be A (First In)
	assert.Equal(t, "A", fifo.SelectVictim())

	fifo.OnRemove("A")
	assert.Equal(t, "B", fifo.SelectVictim())
}

func TestLFUPolicy(t *testing.T) {
	lfu := NewLFU()

	// Add A, B, C
	lfu.OnAdd("A")
	lfu.OnAdd("B")
	lfu.OnAdd("C")

	// Current freq: A=1, B=1, C=1.
	// Access A twice, B once.
	lfu.OnAccess("A") // A=2
	lfu.OnAccess("A") // A=3
	lfu.OnAccess("B") // B=2

	// C is still 1. Victim should be C.
	assert.Equal(t, "C", lfu.SelectVictim())

	lfu.OnRemove("C")

	// Now A=3, B=2. Victim B.
	assert.Equal(t, "B", lfu.SelectVictim())
}

func TestRandomPolicy(t *testing.T) {
	// Use a local, deterministic rand source for reproducible tests
	src := rand.NewSource(42) // Fixed seed for reproducibility
	r := rand.New(src)

	t.Run("RandomSelection", func(t *testing.T) {
		p := newRandomWithRand(r)
		p.OnAdd("A")
		p.OnAdd("B")
		p.OnAdd("C")

		// We can't deterministically test which one is evicted without knowing the rand sequence,
		// but we can ensure it is one of them.
		victim := p.SelectVictim()
		assert.Contains(t, []string{"A", "B", "C"}, victim)

		// Ensure removed item is gone
		p.OnRemove(victim)

		// Verify remaining items are still candidates and the removed item is not selected
		newVictim := p.SelectVictim()
		assert.NotEqual(t, victim, newVictim)
		assert.Contains(t, []string{"A", "B", "C"}, newVictim) // Still one of the original set
	})
}
