package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func BenchmarkCacheHit(b *testing.B) {
	mr := miniredis.RunT(b)
	rc := NewRedisClient(mr.Addr(), "", 0)
	_ = rc.Connect(context.Background())
	cache := NewInstanceCache(rc)
	ctx := context.Background()

	// Pre-populate
	_ = cache.SetInstanceState(ctx, &CachedInstanceState{
		InstanceID: "i-bench",
		State:      "running",
		PublicIP:   "1.2.3.4",
		PrivateIP:  "10.0.0.1",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetInstanceState(ctx, "i-bench")
	}
}

func BenchmarkCacheMiss(b *testing.B) {
	mr := miniredis.RunT(b)
	rc := NewRedisClient(mr.Addr(), "", 0)
	_ = rc.Connect(context.Background())
	cache := NewInstanceCache(rc)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetInstanceState(ctx, "i-nonexistent")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	mr := miniredis.RunT(b)
	rc := NewRedisClient(mr.Addr(), "", 0)
	_ = rc.Connect(context.Background())
	cache := NewInstanceCache(rc)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.SetInstanceState(ctx, &CachedInstanceState{
			InstanceID: fmt.Sprintf("i-%d", i),
			State:      "running",
			PublicIP:   "1.2.3.4",
		})
	}
}

func BenchmarkGetAllInstances(b *testing.B) {
	mr := miniredis.RunT(b)
	rc := NewRedisClient(mr.Addr(), "", 0)
	_ = rc.Connect(context.Background())
	cache := NewInstanceCache(rc)
	ctx := context.Background()

	// Pre-populate 100 instances
	for i := 0; i < 100; i++ {
		_ = cache.SetInstanceState(ctx, &CachedInstanceState{
			InstanceID: fmt.Sprintf("i-%04d", i),
			State:      "running",
			PublicIP:   "1.2.3.4",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetAllInstances(ctx)
	}
}

func BenchmarkNoOpCache(b *testing.B) {
	// Measures overhead when Redis is unavailable (graceful degradation)
	rc := NewRedisClient("", "", 0)
	cache := NewInstanceCache(rc)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetInstanceState(ctx, "i-anything")
	}
}
