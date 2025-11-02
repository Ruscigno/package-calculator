package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sander-remitly/pack-calc/internal/logger"
	"go.uber.org/zap"
)

const (
	// Cache TTL constants
	InitialTTL = 5 * time.Minute
	MaxTTL     = 24 * time.Hour

	// Cache key prefix
	CacheKeyPrefix = "packcalc:"

	// Stats keys
	StatsHitsKey   = "packcalc:stats:hits"
	StatsMissesKey = "packcalc:stats:misses"
)

// CachedResult represents a cached calculation result
type CachedResult struct {
	Items             int           `json:"items"`
	PackSizes         []int         `json:"pack_sizes"`
	Result            map[int]int   `json:"result"`
	TotalItems        int           `json:"total_items"`
	TotalPacks        int           `json:"total_packs"`
	Waste             int           `json:"waste"`
	CalculationTimeMs int64         `json:"calculation_time_ms"`
	CachedAt          time.Time     `json:"cached_at"`
	HitCount          int           `json:"hit_count"`
	CurrentTTL        time.Duration `json:"current_ttl"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
	TotalKeys  int64   `json:"total_keys"`
	MemoryUsed string  `json:"memory_used"`
	Uptime     string  `json:"uptime"`
}

// Cache handles Redis caching operations
type Cache struct {
	client  *redis.Client
	enabled bool
	ctx     context.Context
}

// NewCache creates a new cache instance
func NewCache() *Cache {
	enabled := os.Getenv("REDIS_ENABLED") == "true"

	if !enabled {
		logger.Log.Info("Redis cache is disabled")
		return &Cache{enabled: false, ctx: context.Background()}
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Log.Warn("Failed to connect to Redis. Cache disabled.",
			zap.String("address", redisAddr),
			zap.Error(err),
		)
		return &Cache{enabled: false, ctx: ctx}
	}

	logger.Log.Info("Redis cache enabled", zap.String("address", redisAddr))
	return &Cache{
		client:  client,
		enabled: true,
		ctx:     ctx,
	}
}

// IsEnabled returns whether caching is enabled
func (c *Cache) IsEnabled() bool {
	return c.enabled
}

// generateKey creates a cache key from items and pack sizes
func (c *Cache) generateKey(items int, packSizes []int) string {
	// Sort pack sizes to ensure consistent keys
	sorted := make([]int, len(packSizes))
	copy(sorted, packSizes)
	sort.Ints(sorted)

	// Create a deterministic string representation
	data := fmt.Sprintf("%d:%v", items, sorted)

	// Hash it for a shorter key
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%s%x", CacheKeyPrefix, hash[:16])
}

// Get retrieves a cached result and updates its TTL
func (c *Cache) Get(items int, packSizes []int) (*CachedResult, bool) {
	if !c.enabled {
		return nil, false
	}

	key := c.generateKey(items, packSizes)

	// Get the cached data
	data, err := c.client.Get(c.ctx, key).Bytes()
	if err == redis.Nil {
		// Cache miss
		c.incrementMisses()
		return nil, false
	} else if err != nil {
		log.Printf("Cache get error: %v", err)
		c.incrementMisses()
		return nil, false
	}

	// Deserialize
	var result CachedResult
	if err := json.Unmarshal(data, &result); err != nil {
		log.Printf("Cache unmarshal error: %v", err)
		c.incrementMisses()
		return nil, false
	}

	// Cache hit! Update TTL (double it, up to max)
	result.HitCount++
	newTTL := result.CurrentTTL * 2
	if newTTL > MaxTTL {
		newTTL = MaxTTL
	}
	result.CurrentTTL = newTTL

	// Save back with updated TTL and hit count
	if err := c.set(key, &result, newTTL); err != nil {
		log.Printf("Failed to update cache TTL: %v", err)
	}

	c.incrementHits()
	return &result, true
}

// Set stores a calculation result in cache
func (c *Cache) Set(items int, packSizes []int, result map[int]int, totalItems, totalPacks, waste int, calcTime int64) error {
	if !c.enabled {
		return nil
	}

	key := c.generateKey(items, packSizes)

	cached := &CachedResult{
		Items:             items,
		PackSizes:         packSizes,
		Result:            result,
		TotalItems:        totalItems,
		TotalPacks:        totalPacks,
		Waste:             waste,
		CalculationTimeMs: calcTime,
		CachedAt:          time.Now(),
		HitCount:          0,
		CurrentTTL:        InitialTTL,
	}

	return c.set(key, cached, InitialTTL)
}

// set is an internal method to store data with a specific TTL
func (c *Cache) set(key string, result *CachedResult, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if err := c.client.Set(c.ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// GetStats returns cache statistics
func (c *Cache) GetStats() (*CacheStats, error) {
	if !c.enabled {
		return &CacheStats{}, nil
	}

	// Get hit/miss counts
	hits, _ := c.client.Get(c.ctx, StatsHitsKey).Int64()
	misses, _ := c.client.Get(c.ctx, StatsMissesKey).Int64()

	total := hits + misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	// Get total keys matching our prefix
	keys, err := c.client.Keys(c.ctx, CacheKeyPrefix+"*").Result()
	if err != nil {
		log.Printf("Failed to get cache keys: %v", err)
	}

	// Get memory info
	info, err := c.client.Info(c.ctx, "memory", "server").Result()
	memoryUsed := "N/A"
	uptime := "N/A"

	if err == nil {
		// Parse memory and uptime from info string
		memoryUsed = parseInfoField(info, "used_memory_human")
		uptimeSecs := parseInfoField(info, "uptime_in_seconds")
		if secs, err := strconv.Atoi(uptimeSecs); err == nil {
			uptime = (time.Duration(secs) * time.Second).String()
		}
	}

	return &CacheStats{
		Hits:       hits,
		Misses:     misses,
		HitRate:    hitRate,
		TotalKeys:  int64(len(keys)),
		MemoryUsed: memoryUsed,
		Uptime:     uptime,
	}, nil
}

// Clear removes all cache entries
func (c *Cache) Clear() error {
	if !c.enabled {
		return nil
	}

	keys, err := c.client.Keys(c.ctx, CacheKeyPrefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	if len(keys) > 0 {
		if err := c.client.Del(c.ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete cache keys: %w", err)
		}
	}

	// Reset stats
	c.client.Del(c.ctx, StatsHitsKey, StatsMissesKey)

	return nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	if c.enabled && c.client != nil {
		return c.client.Close()
	}
	return nil
}

// incrementHits increments the cache hit counter
func (c *Cache) incrementHits() {
	c.client.Incr(c.ctx, StatsHitsKey)
}

// incrementMisses increments the cache miss counter
func (c *Cache) incrementMisses() {
	c.client.Incr(c.ctx, StatsMissesKey)
}

// parseInfoField extracts a field value from Redis INFO output
func parseInfoField(info, field string) string {
	lines := []byte(info)
	start := 0
	for i := 0; i < len(lines); i++ {
		if lines[i] == '\n' {
			line := string(lines[start:i])
			start = i + 1

			if len(line) > len(field)+1 && line[:len(field)] == field && line[len(field)] == ':' {
				return line[len(field)+1:]
			}
		}
	}
	return ""
}
