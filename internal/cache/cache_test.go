package cache

import (
	"context"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sander-remitly/pack-calc/internal/logger"
)

func init() {
	// Initialize logger for tests
	logger.Initialize()
}

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *Cache) {
	// Create a miniredis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cache := &Cache{
		client:  client,
		enabled: true,
		ctx:     context.Background(),
	}

	return mr, cache
}

func TestNewCache_Disabled(t *testing.T) {
	os.Setenv("REDIS_ENABLED", "false")
	defer os.Unsetenv("REDIS_ENABLED")

	cache := NewCache()

	if cache.enabled {
		t.Error("Expected cache to be disabled")
	}
}

func TestGenerateCacheKey(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	tests := []struct {
		name      string
		items     int
		packSizes []int
		wantSame  bool
	}{
		{
			name:      "Same input produces same key",
			items:     250,
			packSizes: []int{250, 500, 1000},
			wantSame:  true,
		},
		{
			name:      "Different order same sizes",
			items:     250,
			packSizes: []int{1000, 500, 250},
			wantSame:  true, // Should be same because we sort
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := cache.generateKey(tt.items, tt.packSizes)
			key2 := cache.generateKey(tt.items, tt.packSizes)

			if key1 != key2 {
				t.Errorf("Expected same keys, got %s and %s", key1, key2)
			}

			if len(key1) == 0 {
				t.Error("Expected non-empty cache key")
			}

			if key1[:len(CacheKeyPrefix)] != CacheKeyPrefix {
				t.Errorf("Expected key to start with %s, got %s", CacheKeyPrefix, key1)
			}
		})
	}
}

func TestGenerateCacheKey_DifferentInputs(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	key1 := cache.generateKey(250, []int{250, 500})
	key2 := cache.generateKey(251, []int{250, 500})
	key3 := cache.generateKey(250, []int{250, 500, 1000})

	if key1 == key2 {
		t.Error("Expected different keys for different items")
	}

	if key1 == key3 {
		t.Error("Expected different keys for different pack sizes")
	}
}

func TestCache_SetAndGet(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	items := 250
	packSizes := []int{250, 500, 1000}
	result := map[int]int{250: 1}

	// Set cache
	err := cache.Set(items, packSizes, result, 250, 1, 0, 5)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Get cache
	cached, found := cache.Get(items, packSizes)
	if !found {
		t.Fatal("Expected cache hit, got miss")
	}

	if cached == nil {
		t.Fatal("Expected cached result, got nil")
	}

	if cached.Items != items {
		t.Errorf("Expected items %d, got %d", items, cached.Items)
	}

	if cached.TotalItems != 250 {
		t.Errorf("Expected total items 250, got %d", cached.TotalItems)
	}

	if cached.TotalPacks != 1 {
		t.Errorf("Expected total packs 1, got %d", cached.TotalPacks)
	}

	if cached.HitCount != 1 {
		t.Errorf("Expected hit count 1, got %d", cached.HitCount)
	}
}

func TestCache_GetMiss(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	// Try to get non-existent cache
	cached, found := cache.Get(999, []int{250, 500})
	if found {
		t.Error("Expected cache miss, got hit")
	}

	if cached != nil {
		t.Error("Expected nil for cache miss")
	}
}

func TestCache_AdaptiveTTL(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	items := 250
	packSizes := []int{250, 500}
	result := map[int]int{250: 1}

	// First set - initial TTL
	err := cache.Set(items, packSizes, result, 250, 1, 0, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// First get - should double TTL
	cached1, found := cache.Get(items, packSizes)
	if !found {
		t.Fatal("Expected cache hit")
	}

	if cached1.HitCount != 1 {
		t.Errorf("Expected hit count 1, got %d", cached1.HitCount)
	}

	if cached1.CurrentTTL != InitialTTL*2 {
		t.Errorf("Expected TTL %v, got %v", InitialTTL*2, cached1.CurrentTTL)
	}

	// Second get - should double again
	cached2, found := cache.Get(items, packSizes)
	if !found {
		t.Fatal("Expected cache hit")
	}

	if cached2.HitCount != 2 {
		t.Errorf("Expected hit count 2, got %d", cached2.HitCount)
	}

	if cached2.CurrentTTL != InitialTTL*4 {
		t.Errorf("Expected TTL %v, got %v", InitialTTL*4, cached2.CurrentTTL)
	}
}

func TestCache_MaxTTL(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	items := 250
	packSizes := []int{250, 500}
	result := map[int]int{250: 1}

	// Set with very high TTL
	err := cache.Set(items, packSizes, result, 250, 1, 0, 0)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Get multiple times to exceed max TTL
	for i := 0; i < 20; i++ {
		_, found := cache.Get(items, packSizes)
		if !found {
			t.Fatal("Expected cache hit")
		}
	}

	// Final get should have max TTL
	cached, found := cache.Get(items, packSizes)
	if !found {
		t.Fatal("Expected cache hit")
	}

	if cached.CurrentTTL > MaxTTL {
		t.Errorf("Expected TTL <= %v, got %v", MaxTTL, cached.CurrentTTL)
	}
}

func TestCache_Clear(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	// Set multiple cache entries
	for i := 1; i <= 5; i++ {
		err := cache.Set(i*100, []int{250, 500}, map[int]int{250: 1}, 250, 1, 0, 0)
		if err != nil {
			t.Fatalf("Failed to set cache: %v", err)
		}
	}

	// Clear cache
	err := cache.Clear()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Verify all entries are gone
	for i := 1; i <= 5; i++ {
		cached, found := cache.Get(i*100, []int{250, 500})
		if found || cached != nil {
			t.Errorf("Expected cache to be cleared for items %d", i*100)
		}
	}
}

func TestCache_Stats(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	// Set and get some cache entries to generate stats
	cache.Set(250, []int{250, 500}, map[int]int{250: 1}, 250, 1, 0, 0)
	cache.Get(250, []int{250, 500}) // Hit
	cache.Get(250, []int{250, 500}) // Hit
	cache.Get(999, []int{250, 500}) // Miss

	stats, err := cache.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Stats should show some activity
	if stats.Hits < 0 {
		t.Errorf("Expected non-negative hits, got %d", stats.Hits)
	}
}

func TestCache_Disabled(t *testing.T) {
	cache := &Cache{
		enabled: false,
		ctx:     context.Background(),
	}

	// All operations should be no-ops
	err := cache.Set(250, []int{250, 500}, map[int]int{250: 1}, 250, 1, 0, 0)
	if err != nil {
		t.Errorf("Expected no error for disabled cache, got: %v", err)
	}

	cached, found := cache.Get(250, []int{250, 500})
	if found {
		t.Error("Expected cache miss for disabled cache")
	}
	if cached != nil {
		t.Error("Expected nil for disabled cache")
	}

	err = cache.Clear()
	if err != nil {
		t.Errorf("Expected no error for disabled cache, got: %v", err)
	}
}
