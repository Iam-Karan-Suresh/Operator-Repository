package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func setupTestCache(t *testing.T) (*InstanceCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := NewRedisClient(mr.Addr(), "", 0)
	if err := rc.Connect(context.Background()); err != nil {
		t.Fatalf("failed to connect to miniredis: %v", err)
	}
	return NewInstanceCache(rc), mr
}

func TestGetSetInstanceState(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	// Cache miss
	result := cache.GetInstanceState(ctx, "i-missing")
	if result != nil {
		t.Fatal("expected nil for missing instance")
	}

	// Set and get
	state := &CachedInstanceState{
		InstanceID: "i-12345",
		State:      "running",
		PublicIP:   "1.2.3.4",
		PrivateIP:  "10.0.0.1",
		PublicDNS:  "ec2-1-2-3-4.compute-1.amazonaws.com",
		PrivateDNS: "ip-10-0-0-1.ec2.internal",
		Region:     "us-east-1",
	}
	if err := cache.SetInstanceState(ctx, state); err != nil {
		t.Fatalf("failed to set state: %v", err)
	}

	result = cache.GetInstanceState(ctx, "i-12345")
	if result == nil {
		t.Fatal("expected cached state, got nil")
	}
	if result.InstanceID != "i-12345" {
		t.Errorf("expected instanceID i-12345, got %s", result.InstanceID)
	}
	if result.State != "running" {
		t.Errorf("expected state running, got %s", result.State)
	}
	if result.PublicIP != "1.2.3.4" {
		t.Errorf("expected publicIP 1.2.3.4, got %s", result.PublicIP)
	}
	if result.CachedAt == 0 {
		t.Error("expected CachedAt to be set")
	}
}

func TestDeleteInstance(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	state := &CachedInstanceState{
		InstanceID: "i-delete-me",
		State:      "running",
	}
	_ = cache.SetInstanceState(ctx, state)

	// Verify it exists
	if cache.GetInstanceState(ctx, "i-delete-me") == nil {
		t.Fatal("expected instance to be cached")
	}

	// Delete
	if err := cache.DeleteInstance(ctx, "i-delete-me"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify deletion
	if cache.GetInstanceState(ctx, "i-delete-me") != nil {
		t.Fatal("expected nil after deletion")
	}
}

func TestGetAllInstances(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	// Empty
	all := cache.GetAllInstances(ctx)
	if len(all) != 0 {
		t.Fatalf("expected 0 instances, got %d", len(all))
	}

	// Add multiple
	for _, id := range []string{"i-001", "i-002", "i-003"} {
		_ = cache.SetInstanceState(ctx, &CachedInstanceState{
			InstanceID: id,
			State:      "running",
		})
	}

	all = cache.GetAllInstances(ctx)
	if len(all) != 3 {
		t.Fatalf("expected 3 instances, got %d", len(all))
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	cache, mr := setupTestCache(t)
	ctx := context.Background()

	_ = cache.SetInstanceState(ctx, &CachedInstanceState{
		InstanceID: "i-ttl",
		State:      "running",
	})

	// Verify present
	if cache.GetInstanceState(ctx, "i-ttl") == nil {
		t.Fatal("expected cached state")
	}

	// Fast-forward past TTL
	mr.FastForward(DefaultInstanceTTL + time.Second)

	// Should be expired
	if cache.GetInstanceState(ctx, "i-ttl") != nil {
		t.Fatal("expected nil after TTL expiry")
	}
}

func TestStatsCache(t *testing.T) {
	cache, mr := setupTestCache(t)
	ctx := context.Background()

	// Miss
	if cache.GetStats(ctx) != nil {
		t.Fatal("expected nil stats initially")
	}

	// Set
	stats := &CachedStats{
		ReconciliationCount: 42,
		InstanceCount:       5,
		ApiLatency:          150.5,
		TotalStorage:        100,
	}
	if err := cache.SetStats(ctx, stats); err != nil {
		t.Fatalf("failed to set stats: %v", err)
	}

	// Get
	result := cache.GetStats(ctx)
	if result == nil {
		t.Fatal("expected cached stats")
	}
	if result.ReconciliationCount != 42 {
		t.Errorf("expected reconciliation count 42, got %d", result.ReconciliationCount)
	}
	if result.InstanceCount != 5 {
		t.Errorf("expected instance count 5, got %d", result.InstanceCount)
	}

	// TTL expiry
	mr.FastForward(DefaultStatsTTL + time.Second)
	if cache.GetStats(ctx) != nil {
		t.Fatal("expected nil stats after TTL expiry")
	}
}

func TestNilRedisGracefulDegradation(t *testing.T) {
	// No-op client (addr="")
	rc := NewRedisClient("", "", 0)
	cache := NewInstanceCache(rc)
	ctx := context.Background()

	// All operations should return nil/no-error gracefully
	if cache.GetInstanceState(ctx, "i-xxx") != nil {
		t.Fatal("expected nil from no-op cache")
	}
	if err := cache.SetInstanceState(ctx, &CachedInstanceState{InstanceID: "i-xxx"}); err != nil {
		t.Fatalf("expected no error from no-op set: %v", err)
	}
	if err := cache.DeleteInstance(ctx, "i-xxx"); err != nil {
		t.Fatalf("expected no error from no-op delete: %v", err)
	}
	if cache.GetAllInstances(ctx) != nil {
		t.Fatal("expected nil from no-op list")
	}
	if cache.GetStats(ctx) != nil {
		t.Fatal("expected nil from no-op stats")
	}
}
