package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// RedisClient is a wrapper around the go-redis client.
// It supports a graceful degradation mode (available=false) when Redis is unreachable,
// allowing the operator to continue functioning without cache benefits.
type RedisClient struct {
	client    *redis.Client
	available bool
}

// NewRedisClient creates a new RedisClient instance.
func NewRedisClient(addr string, password string, db int) *RedisClient {
	if addr == "" {
		return &RedisClient{available: false}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisClient{
		client:    rdb,
		available: true,
	}
}

// Connect verifies the connection to Redis. If it fails, the client marks itself as unavailable.
func (r *RedisClient) Connect(ctx context.Context) error {
	if !r.available {
		return nil
	}

	log := logf.FromContext(ctx).WithName("redis-client")

	// Add a timeout for the ping
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		log.Error(err, "Failed to connect to Redis, disabling cache")
		r.available = false
		return err
	}

	log.Info("Successfully connected to Redis cache")
	return nil
}

// Close gracefully shuts down the Redis connection.
func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// IsAvailable returns true if the Redis client is connected and ready.
func (r *RedisClient) IsAvailable() bool {
	return r.available
}

// GetClient returns the underlying go-redis client, or nil if unavailable.
func (r *RedisClient) GetClient() *redis.Client {
	if !r.available {
		return nil
	}
	return r.client
}
