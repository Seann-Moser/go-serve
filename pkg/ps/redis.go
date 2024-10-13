package ps

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/clientpkg"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"sync"
	"time"
)

var _ PubSub[any] = &RedisPubSub[any]{}

type RedisPubSub[T any] struct {
	client         *redis.Client
	defaultChannel string
	ps             *redis.PubSub
}

func RedisPubSubFlags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(clientpkg.GetFlagWithPrefix(prefix, "redis-pub-sub"), pflag.ExitOnError)
	// Define Redis connection flags with the given prefix
	fs.String(clientpkg.GetFlagWithPrefix(prefix, "redis-address"), "localhost:6379", "Redis server address")
	fs.String(clientpkg.GetFlagWithPrefix(prefix, "redis-password"), "", "Redis server password")
	fs.Int(clientpkg.GetFlagWithPrefix(prefix, "redis-db"), 0, "Redis database number")
	fs.String(clientpkg.GetFlagWithPrefix(prefix, "default-channel"), "", "Default Redis channel name")
	return fs
}

func NewRedisPubSubFromFlags[T any](ctx context.Context, prefix string) (*RedisPubSub[T], error) {
	// Retrieve flag values using clientpkg.GetFlagWithPrefix
	redisAddress := viper.GetString(clientpkg.GetFlagWithPrefix(prefix, "redis-address"))
	redisPassword := viper.GetString(clientpkg.GetFlagWithPrefix(prefix, "redis-password"))
	redisDB := viper.GetInt(clientpkg.GetFlagWithPrefix(prefix, "redis-db"))
	defaultChannel := viper.GetString(clientpkg.GetFlagWithPrefix(prefix, "default-channel"))

	// Validate required flags
	if redisAddress == "" {
		return nil, fmt.Errorf("redis-address is required")
	}

	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: redisPassword,
		DB:       redisDB,
	})
	t := time.NewTicker(time.Second)
	tmpCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := client.Ping(ctx)
	for err != nil {
		select {
		case <-tmpCtx.Done():
			cancel()
			t.Stop()
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		case <-t.C:
			err = client.Ping(ctx)
		}
	}

	return &RedisPubSub[T]{
		client:         client,
		defaultChannel: defaultChannel,
	}, nil
}

// Ping sends a PING command to the Redis server to check connectivity.
// It returns an error if the server does not respond within the specified timeout.
func (r *RedisPubSub[T]) Ping(ctx context.Context, timeout time.Duration) error {
	// Create a context with the specified timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Send the PING command
	if r.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	ping := r.client.Ping(ctx)
	if ping == nil {
		return fmt.Errorf("status is nil")
	}
	err := ping.Err()
	if err != nil {
		return fmt.Errorf("failed to ping Redis server: %w", err)
	}

	return nil
}

// Publish publishes messages to the specified Redis channel.
func (r *RedisPubSub[T]) Publish(ctx context.Context, channel string, data chan *T, workers int) error {
	if channel == "" {
		if r.defaultChannel == "" {
			return fmt.Errorf("channel is required")
		}
		channel = r.defaultChannel
	}

	var wg sync.WaitGroup
	workerCount := workers
	if workerCount <= 0 {
		workerCount = 1
	}

	// Create worker goroutines to process messages.
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for msg := range data {
				select {
				case <-ctx.Done():
					return
				default:
					// Marshal the message to JSON
					b, err := json.Marshal(msg)
					if err != nil {
						// Handle marshaling error (e.g., log it)
						continue
					}

					// Publish the message to Redis
					err = r.client.Publish(ctx, channel, b).Err()
					if err != nil {
						// Handle publishing error (e.g., log it)
						continue
					}
				}
			}
		}(i)
	}

	// Wait for all workers to finish
	wg.Wait()
	return nil
}

// Subscribe subscribes to a given channel (topic) and returns a Subscription.
func (r *RedisPubSub[T]) Subscribe(ctx context.Context, channel string) (*Subscription[T], error) {
	if channel == "" {
		if r.defaultChannel == "" {
			return nil, fmt.Errorf("channel is required")
		}
		channel = r.defaultChannel
	}

	// Create a Redis PubSub instance
	r.ps = r.client.Subscribe(ctx, channel)

	// Wait for confirmation that subscription is created
	_, err := r.ps.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to channel '%s': %w", channel, err)
	}

	// Create a channel to receive SubscriptionData
	dataCh := make(chan *SubscriptionData[T], 100) // Buffered to prevent blocking

	// Start a goroutine to listen for messages
	go func() {
		defer close(dataCh)
		ch := r.ps.Channel()

		for msg := range ch {
			var data T
			err := json.Unmarshal([]byte(msg.Payload), &data)
			if err != nil {
				// Handle unmarshaling error (e.g., log it)
				continue
			}

			subData := &SubscriptionData[T]{
				data: &data,
				Ack: func(ctx context.Context) error {
					// Redis Pub/Sub does not support acknowledgments
					// This is a no-op in this implementation
					return nil
				},
				Nack: func(ctx context.Context) error {
					// Redis Pub/Sub does not support negative acknowledgments
					// This is a no-op in this implementation
					return nil
				},
			}

			select {
			case dataCh <- subData:
			case <-ctx.Done():
				return
			}
		}
	}()

	subscriptionObj := &Subscription[T]{
		Name:      channel,
		c:         dataCh,
		closeOnce: sync.Once{},
		closeFunc: func() {
			// Unsubscribe from the Redis channel
			_ = r.ps.Unsubscribe(ctx, channel)
			r.ps.Close()
			r.ps = nil
			ctxLogger.Error(ctx, "closed subscription")
		},
	}

	return subscriptionObj, nil
}

// Close closes the RedisPubSub client.
func (r *RedisPubSub[T]) Close() error {
	if r.ps != nil {
		r.ps.Close()
	}
	r.client.Close()
	return nil
}
