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

func NewLocalCache(eviction time.Duration) LocalCache {
	cache, err := bigcache.New(context.Background(), bigcache.DefaultConfig(eviction))
	if err != nil {
		panic("NewLocalCache: " + err.Error())
	}
	return LocalCache{BigCache: cache}
}

func (c *LocalCache) Get(key string, v interface{}) error {
	data, err := c.BigCache.Get(key)
	if err != nil {
		return err
	}
	return Deserialize(data, v)
}

func (c *LocalCache) Set(key string, v interface{}) {
	data, err := Serialize(v)
	if err == nil {
		err := c.BigCache.Set(key, data)
		if err != nil {
			fmt.Printf("key: %s , cache set error: %s", key, err.Error())
		}
	}
}

func (c *LocalCache) Delete(key string) {
	_ = c.BigCache.Delete(key)
}
