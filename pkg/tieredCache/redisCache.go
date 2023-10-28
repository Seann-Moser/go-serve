package tieredCache

import (
	"context"
	"encoding/json"
	redis "github.com/redis/go-redis/v9"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"time"
)

var _ Cache = &RedisCache{}

type RedisCache struct {
	cacher          *redis.Client
	defaultDuration time.Duration
}

func RedisFlags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(prefix+"redis", pflag.ExitOnError)
	fs.String(prefix+"redis-addr", "", "")
	fs.String(prefix+"redis-pass", "", "")
	fs.String(prefix+"redis-user", "", "")

	fs.Duration(prefix+"redis-cleanup-duration", 1*time.Minute, "")

	return fs
}
func NewRedisCacheFromFlags(prefix string) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Network:  "",
		Addr:     viper.GetString(prefix + "redis-addr"),
		Username: viper.GetString(prefix + "redis-user"),
		Password: viper.GetString(prefix + "redis-pass"),
	})

	return NewRedisCache(rdb, viper.GetDuration(prefix+"redis-cleanup-duration"))
}

func NewRedisCache(cacher *redis.Client, defaultDuration time.Duration) *RedisCache {
	return &RedisCache{
		cacher:          cacher,
		defaultDuration: defaultDuration,
	}
}

func (c *RedisCache) SetCache(ctx context.Context, key string, item interface{}) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	stats := c.cacher.Set(ctx, key, data, c.defaultDuration)
	return stats.Err()
}

func (c *RedisCache) GetCache(ctx context.Context, key string) ([]byte, error) {
	return c.cacher.Get(ctx, key).Bytes()
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.cacher.Ping(ctx).Err()
}
