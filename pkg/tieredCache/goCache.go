package tieredCache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/patrickmn/go-cache"
)

var _ Cache = &GoCache{}

type GoCache struct {
	defaultDuration time.Duration
	cacher          *cache.Cache
}

func NewGoCache(cacher *cache.Cache, defaultDuration time.Duration) *GoCache {
	return &GoCache{
		cacher:          cacher,
		defaultDuration: defaultDuration,
	}
}

func (c *GoCache) SetCache(ctx context.Context, key string, item interface{}) error {
	c.cacher.Set(key, item, c.defaultDuration)
	return nil
}

func (c *GoCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	if data, found := c.cacher.Get(key); found {
		return nil, ErrCacheMiss
	} else {
		return json.Marshal(data)
	}
}
