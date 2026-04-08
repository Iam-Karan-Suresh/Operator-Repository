package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	DefaultInstanceTTL = 60 * time.Second
	DefaultStatsTTL    = 5 * time.Second
	instanceKeyPrefix  = "instance:"
	statsKey           = "global:stats"
)

var (
	cacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ec2_operator_cache_hits_total",
		Help: "Total number of cache hits in Redis",
	})
	cacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ec2_operator_cache_misses_total",
		Help: "Total number of cache misses in Redis",
	})
)

func init() {
	metrics.Registry.MustRegister(cacheHits, cacheMisses)
}

// CachedInstanceState represents the data stored in Redis for an EC2 instance.
type CachedInstanceState struct {
	InstanceID string `json:"instanceID"`
	State      string `json:"state"`
	PublicIP   string `json:"publicIP,omitempty"`
	PrivateIP  string `json:"privateIP,omitempty"`
	PublicDNS  string `json:"publicDNS,omitempty"`
	PrivateDNS string `json:"privateDNS,omitempty"`
	Region     string `json:"region,omitempty"`
	CachedAt   int64  `json:"cachedAt"`
}

type CachedStats struct {
	ReconciliationCount int64   `json:"reconciliationCount"`
	InstanceCount       int     `json:"instanceCount"`
	ApiLatency          float64 `json:"apiLatency"`
	TotalStorage        int64   `json:"totalStorage"`
}

// InstanceCache provides high-level operations for caching EC2 instance states.
type InstanceCache struct {
	rc *RedisClient
}

func NewInstanceCache(rc *RedisClient) *InstanceCache {
	return &InstanceCache{rc: rc}
}

// GetInstanceState retrieves the state of an instance from Redis.
// Returns nil if the key is not found or Redis is unavailable (graceful degradation).
func (c *InstanceCache) GetInstanceState(ctx context.Context, instanceID string) *CachedInstanceState {
	if !c.rc.IsAvailable() {
		return nil
	}

	key := instanceKeyPrefix + instanceID
	val, err := c.rc.GetClient().Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			cacheMisses.Inc()
		} else {
			logf.FromContext(ctx).Error(err, "Redis Get error", "key", key)
		}
		return nil
	}

	var state CachedInstanceState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to unmarshal cached state", "key", key)
		return nil
	}

	cacheHits.Inc()
	return &state
}

// SetInstanceState caches the state of an instance in Redis with a default TTL.
func (c *InstanceCache) SetInstanceState(ctx context.Context, state *CachedInstanceState) error {
	if !c.rc.IsAvailable() {
		return nil
	}

	state.CachedAt = time.Now().Unix()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	key := instanceKeyPrefix + state.InstanceID
	if err := c.rc.GetClient().Set(ctx, key, data, DefaultInstanceTTL).Err(); err != nil {
		logf.FromContext(ctx).Error(err, "Redis Set error", "key", key)
		return err
	}
	return nil
}

// DeleteInstance removes an instance's state from the cache.
func (c *InstanceCache) DeleteInstance(ctx context.Context, instanceID string) error {
	if !c.rc.IsAvailable() {
		return nil
	}

	key := instanceKeyPrefix + instanceID
	return c.rc.GetClient().Del(ctx, key).Err()
}

// GetAllInstances retrieves all cached instances. Useful for Dashboard list operations.
func (c *InstanceCache) GetAllInstances(ctx context.Context) []CachedInstanceState {
	if !c.rc.IsAvailable() {
		return nil
	}

	client := c.rc.GetClient()
	var cursor uint64
	var keys []string
	var err error

	// SCAN for all instance keys
	for {
		var batch []string
		batch, cursor, err = client.Scan(ctx, cursor, instanceKeyPrefix+"*", 100).Result()
		if err != nil {
			logf.FromContext(ctx).Error(err, "Redis Scan error")
			return nil
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return nil
	}

	// MGET all found keys
	vals, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		logf.FromContext(ctx).Error(err, "Redis MGet error")
		return nil
	}

	instances := make([]CachedInstanceState, 0, len(vals))
	for _, val := range vals {
		if strVal, ok := val.(string); ok {
			var state CachedInstanceState
			if err := json.Unmarshal([]byte(strVal), &state); err == nil {
				instances = append(instances, state)
			}
		}
	}

	return instances
}

// GetStats retrieves global stats from the cache.
func (c *InstanceCache) GetStats(ctx context.Context) *CachedStats {
	if !c.rc.IsAvailable() {
		return nil
	}

	val, err := c.rc.GetClient().Get(ctx, statsKey).Result()
	if err != nil {
		return nil
	}

	var stats CachedStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return nil
	}

	return &stats
}

// SetStats caches global stats with a short TTL.
func (c *InstanceCache) SetStats(ctx context.Context, stats *CachedStats) error {
	if !c.rc.IsAvailable() {
		return nil
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	return c.rc.GetClient().Set(ctx, statsKey, data, DefaultStatsTTL).Err()
}
