package tieredCache

import (
	"context"
	"encoding/json"
	"errors"

	"go.uber.org/multierr"
)

var (
	ErrCacheMiss = errors.New("cache missed")
)

type Cache interface {
	SetCache
	GetCache
	Ping(ctx context.Context) error
}
type SetCache interface {
	SetCache(ctx context.Context, key string, item interface{}) error
}
type GetCache interface {
	GetCache(ctx context.Context, key string) ([]byte, error)
}

func Set[T any](ctx context.Context, key string, data T) error {
	return GetCacheFromContext(ctx).SetCache(ctx, key, data)
}

func Get[T any](ctx context.Context, key string) (*T, error) {
	data, err := GetCacheFromContext(ctx).GetCache(ctx, key)
	if err != nil {
		return nil, err
	}
	var output T
	err = json.Unmarshal(data, output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

const (
	CTX_CACHE = "cache_ctx"
)

func ContextWithCache(ctx context.Context, cache Cache) context.Context {
	return context.WithValue(ctx, CTX_CACHE, cache) //nolint:staticcheck
}

func GetCacheFromContext(ctx context.Context) Cache {
	if ctx == nil {
		return &GoCache{}
	}
	cache := ctx.Value(CTX_CACHE)
	if cache == nil {
		return &GoCache{}
	}
	return cache.(Cache)
}

var _ Cache = &TieredCache{}

type TieredCache struct {
	cachePool []Cache
	getter    GetCache
}

func NewTieredCache(setter GetCache, cacheList ...Cache) Cache {
	return &TieredCache{
		cachePool: cacheList,
		getter:    setter,
	}
}
func (t *TieredCache) Ping(ctx context.Context) error {
	return nil
}

func (t *TieredCache) SetCache(ctx context.Context, key string, item interface{}) error {
	var err error
	for _, c := range t.cachePool {
		err = multierr.Combine(err, c.SetCache(ctx, key, item))
	}
	return err
}

func (t *TieredCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	var missedCacheList []Cache
	var v []byte
	var err error
	defer func() {
		for _, c := range missedCacheList {
			_ = c.SetCache(ctx, key, v)
		}
	}()
	for _, c := range t.cachePool {
		if v, err = c.GetCache(ctx, key); err != nil && v != nil {
			return v, err
		} else {
			missedCacheList = append(missedCacheList, c)
		}
	}

	v, err = t.getter.GetCache(ctx, key)
	if err != nil {
		missedCacheList = []Cache{}
		return nil, err
	}
	return v, nil
}
