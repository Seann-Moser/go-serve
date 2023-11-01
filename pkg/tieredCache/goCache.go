package tieredCache

import (
	"context"
	"encoding/json"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"time"

	"github.com/patrickmn/go-cache"
)

var _ Cache = &GoCache{}

type GoCache struct {
	defaultDuration time.Duration
	cacher          *cache.Cache
}

func GoCacheFlags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix+"gocache", pflag.ExitOnError)
	fs.Duration(prefix+"gocache-default-duration", 1*time.Minute, "")
	fs.Duration(prefix+"gocache-cleanup-duration", 1*time.Minute, "")

	return fs
}
func NewGoCacheFromFlags(prefix string) *GoCache {
	return NewGoCache(cache.New(viper.GetDuration(prefix+"gocache-cleanup-duration"), viper.GetDuration(prefix+"memcache-default-duration")), viper.GetDuration(prefix+"memcache-default-duration"))
}

func NewGoCache(cacher *cache.Cache, defaultDuration time.Duration) *GoCache {
	return &GoCache{
		cacher:          cacher,
		defaultDuration: defaultDuration,
	}
}

func (c *GoCache) Ping(ctx context.Context) error {
	return nil
}

func (c *GoCache) SetCache(ctx context.Context, key string, item interface{}) error {
	c.cacher.Set(key, item, c.defaultDuration)
	return nil
}

func (c *GoCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	if data, found := c.cacher.Get(key); !found {
		return nil, ErrCacheMiss
	} else {
		switch v := data.(type) {
		case string:
			return []byte(v), nil
		default:
			return json.Marshal(data)
		}
	}
}
