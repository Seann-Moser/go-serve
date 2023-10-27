package tieredCache

import (
	"context"
)

var _ Cache = &MemCache{}

type MemCache struct {
}

func (c *MemCache) SetCache(ctx context.Context, key string, item interface{}) error {
	return nil
}

func (c *MemCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
