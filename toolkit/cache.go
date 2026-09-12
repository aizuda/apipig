package toolkit

import (
	"context"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"time"
)

type LocalCache struct {
	BigCache *bigcache.BigCache
}

// NewLocalCache creates a local cache. Use NewLocalCacheWithError to handle
// initialization failures without panicking.
func NewLocalCache(eviction time.Duration) LocalCache {
	cache, err := NewLocalCacheWithError(eviction)
	if err != nil {
		panic(err)
	}
	return *cache
}

func NewLocalCacheWithError(eviction time.Duration) (*LocalCache, error) {
	cache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(eviction))
	if err != nil {
		return nil, fmt.Errorf("create local cache: %w", err)
	}
	return &LocalCache{BigCache: cache}, nil
}

func (c *LocalCache) Get(key string, v any) error {
	if c == nil || c.BigCache == nil {
		return fmt.Errorf("local cache is not initialized")
	}
	data, err := c.BigCache.Get(key)
	if err != nil {
		return err
	}
	return Deserialize(data, v)
}

func (c *LocalCache) Set(key string, v any) error {
	if c == nil || c.BigCache == nil {
		return fmt.Errorf("local cache is not initialized")
	}
	data, err := Serialize(v)
	if err != nil {
		return fmt.Errorf("serialize cache value: %w", err)
	}
	if err := c.BigCache.Set(key, data); err != nil {
		return fmt.Errorf("set cache key %q: %w", key, err)
	}
	return nil
}

func (c *LocalCache) Delete(key string) error {
	if c == nil || c.BigCache == nil {
		return fmt.Errorf("local cache is not initialized")
	}
	return c.BigCache.Delete(key)
}
