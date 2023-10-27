package tieredCache

import (
	"context"
)

var _ Cache = &RedisCache{}

type RedisCache struct {
}

func (c *RedisCache) SetCache(ctx context.Context, key string, item interface{}) error {
	return nil
}

func (c *RedisCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
