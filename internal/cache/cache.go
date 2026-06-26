// Package cache is a tiny Redis-backed cache with a no-op fallback.
//
// When REDIS_URL is set the app caches the expensive "top sellers" report in
// Redis; otherwise it degrades to a no-op cache (every call is a miss) so the
// rest of the demo still runs without Redis.
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is the minimal interface the app needs.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, val string, ttl time.Duration)
	Enabled() bool
}

// New returns a Redis cache when url is non-empty and reachable, otherwise a
// no-op cache. It never fails the app: a bad/absent Redis just means no caching.
func New(ctx context.Context, url string) Cache {
	if url == "" {
		return noop{}
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return noop{}
	}
	client := redis.NewClient(opt)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return noop{}
	}
	return &redisCache{client: client}
}

type redisCache struct{ client *redis.Client }

func (c *redisCache) Get(ctx context.Context, key string) (string, bool) {
	v, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

func (c *redisCache) Set(ctx context.Context, key, val string, ttl time.Duration) {
	_ = c.client.Set(ctx, key, val, ttl).Err()
}

func (c *redisCache) Enabled() bool { return true }

type noop struct{}

func (noop) Get(context.Context, string) (string, bool)        { return "", false }
func (noop) Set(context.Context, string, string, time.Duration) {}
func (noop) Enabled() bool                                      { return false }
