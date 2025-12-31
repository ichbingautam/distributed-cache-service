package policy

import (
	"container/list"
	"sync"
)

// FIFOPolicy implements the First-In-First-Out (FIFO) eviction strategy.
type FIFOPolicy struct {
	mu    sync.Mutex
	order *list.List
	items map[string]*list.Element
}

// NewFIFO creates a new FIFO policy instance.
func NewFIFO() *FIFOPolicy {
	return &FIFOPolicy{
		order: list.New(),
		items: make(map[string]*list.Element),
	}
}

func (p *FIFOPolicy) OnAccess(key string) {
	// FIFO does not change order on access
}

func (p *FIFOPolicy) OnAdd(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// If exists, do nothing regarding order (it preserves first insertion time)
	// But usually, overwrite might update value, but key "age" remains?
	// Strictly FIFO is based on insertion. If we update, does it reset?
	// Usually invalidataion/update might reset or keep.
	// We will keep original insertion order unless removed.
	if _, ok := p.items[key]; ok {
		return
	}
	elem := p.order.PushBack(key)
	p.items[key] = elem
}

func (p *FIFOPolicy) OnRemove(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if elem, ok := p.items[key]; ok {
		p.order.Remove(elem)
		delete(p.items, key)
	}
}

func (p *FIFOPolicy) SelectVictim() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	elem := p.order.Front() // The oldest element
	if elem != nil {
		return elem.Value.(string)
	}
	return ""
}
