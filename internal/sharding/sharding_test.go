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

func TestMap_DistributionSkew(t *testing.T) {
	// Test with 1 virtual node (High Skew expected)
	m1 := New(1, nil)
	nodes := []string{"node1", "node2", "node3", "node4", "node5"}
	m1.Add(nodes...)

	// Note: Virtual Nodes are NOT Replicas.
	// - Virtual Nodes: Improve distribution (Consistent Hashing).
	// - Replicas (Raft): Provide Fault Tolerance (Redundancy).
	// Here we test distribution logic only.

	counts1 := make(map[string]int)
	for i := 0; i < 1000; i++ {
		key := "key_" + strconv.Itoa(i)
		node := m1.Get(key)
		counts1[node]++
	}

	// Test with 100 virtual nodes (Low Skew expected)
	m100 := New(100, nil)
	m100.Add(nodes...)

	counts100 := make(map[string]int)
	for i := 0; i < 1000; i++ {
		key := "key_" + strconv.Itoa(i)
		node := m100.Get(key)
		counts100[node]++
	}

	// Calculate Standard Deviation for both
	stdDev1 := calculateStdDev(counts1, 1000, len(nodes))
	stdDev100 := calculateStdDev(counts100, 1000, len(nodes))

	t.Logf("StdDev (1 vNode): %.2f", stdDev1)
	t.Logf("StdDev (100 vNodes): %.2f", stdDev100)

	if stdDev100 >= stdDev1 {
		t.Errorf("Expected 100 vNodes to have lower skew (stdDev %.2f) than 1 vNode (stdDev %.2f)", stdDev100, stdDev1)
	}
}

func calculateStdDev(counts map[string]int, total, n int) float64 {
	mean := float64(total) / float64(n)
	var sumSquares float64
	for _, count := range counts {
		diff := float64(count) - mean
		sumSquares += diff * diff
	}
	// Add 0s for nodes that got no keys
	missing := n - len(counts)
	for i := 0; i < missing; i++ {
		diff := 0 - mean
		sumSquares += diff * diff
	}
	return (sumSquares / float64(n)) // Simplified variance (not sqrt for comparison but named stddev for clarity)
}
