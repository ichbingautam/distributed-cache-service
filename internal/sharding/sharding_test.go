package sharding

import (
	"strconv"
	"testing"
)

func TestMap_Add(t *testing.T) {
	m := New(3, nil)
	m.Add("node1", "node2")

	key := "my_key"
	node := m.Get(key)
	if node == "" {
		t.Fatal("expected to get a node")
	}
}

func TestMap_Consistency(t *testing.T) {
	m := New(3, nil)
	m.Add("node1", "node2", "node3")

	key := "stable_key"
	node1 := m.Get(key)

	// Adding a new node shouldn't change the mapping for MOST keys,
	// but for a single key it might.
	// However, if we remove the node we got, we should get another one.
	// Let's test stability: call Get multiple times
	if m.Get(key) != node1 {
		t.Fatalf("hashing should be consistent")
	}
}

func TestMap_Distribution(t *testing.T) {
	// Simple statistical test (loose)
	m := New(20, nil) // 20 virtual nodes
	nodes := []string{"nodeA", "nodeB", "nodeC"}
	m.Add(nodes...)

	counts := make(map[string]int)
	for i := 0; i < 1000; i++ {
		key := "key_" + strconv.Itoa(i)
		node := m.Get(key)
		counts[node]++
	}

	for _, n := range nodes {
		if counts[n] == 0 {
			t.Errorf("node %s got 0 keys, bad distribution", n)
		}
	}
}
