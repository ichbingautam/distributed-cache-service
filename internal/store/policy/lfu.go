package policy

import (
	"container/heap"
	"sync"
)

// lfuItem represents an item in the priority queue.
type lfuItem struct {
	key       string
	frequency int
	index     int // The index of the item in the heap.
}

// PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*lfuItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the lowest frequency (Min-Heap)
	return pq[i].frequency < pq[j].frequency
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*lfuItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// LFUPolicy implements the Least Frequently Used (LFU) eviction strategy.
type LFUPolicy struct {
	mu    sync.Mutex
	pq    PriorityQueue
	items map[string]*lfuItem
}

// NewLFU creates a new LFU policy instance.
func NewLFU() *LFUPolicy {
	return &LFUPolicy{
		pq:    make(PriorityQueue, 0),
		items: make(map[string]*lfuItem),
	}
}

func (p *LFUPolicy) OnAccess(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if item, ok := p.items[key]; ok {
		item.frequency++
		heap.Fix(&p.pq, item.index)
	}
}

func (p *LFUPolicy) OnAdd(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if item, ok := p.items[key]; ok {
		item.frequency++
		heap.Fix(&p.pq, item.index)
		return
	}
	item := &lfuItem{
		key:       key,
		frequency: 1,
	}
	heap.Push(&p.pq, item)
	p.items[key] = item
}

func (p *LFUPolicy) OnRemove(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if item, ok := p.items[key]; ok {
		heap.Remove(&p.pq, item.index)
		delete(p.items, key)
	}
}

func (p *LFUPolicy) SelectVictim() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pq) == 0 {
		return ""
	}
	// Peek the min item
	item := p.pq[0]
	return item.key
}
