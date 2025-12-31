package store

import (
	"fmt"
	"testing"
)

func BenchmarkStore_Set(b *testing.B) {
	s := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		s.Set(key, "value", 0)
	}
}

func BenchmarkStore_Get(b *testing.B) {
	s := New()
	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		s.Set(key, "value", 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			s.Get(key)
			i++
		}
	})
}
