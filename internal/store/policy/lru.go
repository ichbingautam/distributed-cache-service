package policy

import (
	"container/list"
	"sync"
)

// LRUPolicy implements the Least Recently Used (LRU) eviction strategy.
type LRUPolicy struct {
	mu    sync.Mutex
	order *list.List
	items map[string]*list.Element
}

// NewLRU creates a new LRU policy instance.
func NewLRU() *LRUPolicy {
	return &LRUPolicy{
		order: list.New(),
		items: make(map[string]*list.Element),
	}
}

func (p *LRUPolicy) OnAccess(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if elem, ok := p.items[key]; ok {
		p.order.MoveToFront(elem)
	}
}

func (p *LRUPolicy) OnAdd(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// If already exists, just update access
	if elem, ok := p.items[key]; ok {
		p.order.MoveToFront(elem)
		return
	}
	elem := p.order.PushFront(key)
	p.items[key] = elem
}

func (p *LRUPolicy) OnRemove(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if elem, ok := p.items[key]; ok {
		p.order.Remove(elem)
		delete(p.items, key)
	}
}

func (p *LRUPolicy) SelectVictim() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	elem := p.order.Back()
	if elem != nil {
		return elem.Value.(string)
	}
	return ""
}
