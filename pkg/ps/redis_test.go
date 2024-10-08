// redis_pubsub_test.go
package ps

import (
	"context"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/clientpkg"
	"sync"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// setupViper initializes viper with the given flags and default values for testing.
func setupViper(prefix string, flags *pflag.FlagSet, defaults map[string]interface{}) {
	for key, value := range defaults {
		viper.Set(clientpkg.GetFlagWithPrefix(prefix, key), value)
	}
	viper.BindPFlags(flags)
}

// TestRedisPublishAndSubscribe verifies that messages published to a channel are received by subscribers.
func TestRedisPublishAndSubscribe(t *testing.T) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "testRedis-pubsub"
	flags := pflag.NewFlagSet("testRedis-pubsub", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "testRedis-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		t.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		t.Skipf("Skipping TestRedisPublishAndSubscribe because Redis ping failed: %v", err)
	}

	// Subscribe to a channel
	subscription, err := pubsub.Subscribe(ctx, "testRedis-channel")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer subscription.Close(ctx)

	// Create a channel to send messages
	messageChannel := make(chan *TestMessage)

	// Start publishing messages
	go func() {
		defer close(messageChannel)
		for i := 1; i <= 5; i++ {
			messageChannel <- &TestMessage{Content: fmt.Sprintf("Message %d", i)}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Start publishing
	go func() {
		err := pubsub.Publish(ctx, "testRedis-channel", messageChannel, 2)
		if err != nil {
			t.Errorf("Publish failed: %v", err)
		}
	}()

	// Collect received messages
	var received []string
	for i := 1; i <= 5; i++ {
		msg, err := subscription.BPop(ctx)
		if err != nil {
			t.Errorf("Failed to receive message: %v", err)
			break
		}
		received = append(received, msg.data.Content)
		msg.Ack(ctx)
	}

	// Verify all messages are received.
	expected := []string{
		"Message 1",
		"Message 2",
		"Message 3",
		"Message 4",
		"Message 5",
	}

	if len(received) != len(expected) {
		t.Fatalf("Expected %d messages, got %d", len(expected), len(received))
	}

	for i, msg := range received {
		if msg != expected[i] {
			t.Errorf("Expected message '%s', got '%s'", expected[i], msg)
		}
	}
}

// TestRedisMultipleSubscribers ensures that multiple subscribers receive all published messages.
func TestRedisMultipleSubscribers(t *testing.T) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "testRedis-pubsub-multi"
	flags := pflag.NewFlagSet("testRedis-pubsub-multi", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "testRedis-multi-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		t.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		t.Skipf("Skipping TestRedisMultipleSubscribers because Redis ping failed: %v", err)
	}

	topic := "testRedis-multi-channel"
	subscriberCount := 3
	messageCount := 10

	// Create multiple subscribers.
	var subscriptions []*Subscription[TestMessage]
	for i := 0; i < subscriberCount; i++ {
		sub, err := pubsub.Subscribe(ctx, topic)
		if err != nil {
			t.Fatalf("Failed to subscribe %d: %v", i, err)
		}
		defer sub.Close(ctx)
		subscriptions = append(subscriptions, sub)
	}

	messageChannel := make(chan *TestMessage)

	// Start publishing messages.
	go func() {
		defer close(messageChannel)
		for i := 1; i <= messageCount; i++ {
			messageChannel <- &TestMessage{Content: fmt.Sprintf("Msg %d", i)}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Start publishing
	go func() {
		err := pubsub.Publish(ctx, topic, messageChannel, 4)
		if err != nil {
			t.Errorf("Publish failed: %v", err)
		}
	}()

	// Collect messages for each subscriber.
	var wg sync.WaitGroup
	wg.Add(subscriberCount)

	for _, sub := range subscriptions {
		go func(s *Subscription[TestMessage]) {
			defer wg.Done()
			for i := 0; i < messageCount; i++ {
				msg, err := s.BPop(ctx)
				if err != nil {
					t.Errorf("Subscriber failed to receive message: %v", err)
					return
				}
				if msg.data.Content != fmt.Sprintf("Msg %d", i+1) {
					t.Errorf("Expected 'Msg %d', got '%s'", i+1, msg.data.Content)
				}
				msg.Ack(ctx)
			}
		}(sub)
	}

	wg.Wait()
}

// TestRedisSubscriptionClose checks that closing a subscription stops message delivery.
func TestRedisSubscriptionClose(t *testing.T) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "testRedis-pubsub-close"
	flags := pflag.NewFlagSet("testRedis-pubsub-close", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "testRedis-close-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		t.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		t.Skipf("Skipping TestRedisSubscriptionClose because Redis ping failed: %v", err)
	}

	topic := "testRedis-close-channel"
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	messageChannel := make(chan *TestMessage)

	// Start publishing messages.
	go func() {
		for i := 1; i <= 5; i++ {
			messageChannel <- &TestMessage{Content: fmt.Sprintf("CloseMsg %d", i)}
		}
		close(messageChannel)
	}()

	// Start publishing
	go func() {
		err := pubsub.Publish(ctx, topic, messageChannel, 2)
		if err != nil {
			t.Errorf("Publish failed: %v", err)
		}
	}()

	// Receive first two messages.
	for i := 1; i <= 2; i++ {
		msg, err := subscription.BPop(ctx)
		if err != nil {
			t.Errorf("Failed to receive message: %v", err)
		}
		msg.Ack(ctx)
	}

	// Close the subscription.
	subscription.Close(ctx)

	// Allow some time for the closure to propagate.
	time.Sleep(100 * time.Millisecond)

	// Attempt to receive another message.
	msg, err := subscription.BPop(ctx)
	if err == nil {
		t.Errorf("Expected error after closing subscription, but got none. Received message: %v", msg.data.Content)
	}
}

// TestRedisSubscriptionCloseMultipleTimes ensures that closing a subscription multiple times does not cause a panic.
func TestRedisSubscriptionCloseMultipleTimes(t *testing.T) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "testRedis-pubsub-close-multi"
	flags := pflag.NewFlagSet("testRedis-pubsub-close-multi", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "testRedis-close-multi-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		t.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	topic := "testRedis-close-multi-channel"
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// First closure should succeed.
	subscription.Close(ctx)

	// Subsequent closures should have no effect and not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Recovered from panic: %v", r)
		}
	}()

	subscription.Close(ctx) // Should not panic.
	subscription.Close(ctx) // Should not panic.
}

// TestRedisPubSubClose ensures that closing the Pub/Sub system prevents further publishing and subscriptions.
func TestRedisPubSubClose(t *testing.T) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "testRedis-pubsub-global-close"
	flags := pflag.NewFlagSet("testRedis-pubsub-global-close", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "testRedis-global-close-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		t.Fatalf("Failed to create RedisPubSub: %v", err)
	}

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		t.Skipf("Skipping TestRedisPubSubClose because Redis ping failed: %v", err)
	}

	topic := "testRedis-global-close-channel"

	// Close the PubSub system.
	err = pubsub.Close()
	if err != nil {
		t.Fatalf("Failed to close PubSub: %v", err)
	}

	// Attempt to subscribe.
	_, err = pubsub.Subscribe(ctx, topic)
	if err == nil {
		t.Errorf("Expected error when subscribing to closed PubSub, but got none")
	}

	// Attempt to publish.
	messageChannel := make(chan *TestMessage)
	go func() {
		messageChannel <- &TestMessage{Content: "Should Fail"}
		close(messageChannel)
	}()
	err = pubsub.Publish(ctx, topic, messageChannel, 2)
	if err == nil {
		t.Errorf("Expected error when publishing to closed PubSub, but got none")
	}
}

// TestRedisPopWithTimeout validates the Pop method with a timeout.
func TestRedisPopWithTimeout(t *testing.T) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "testRedis-pubsub-pop-timeout"
	flags := pflag.NewFlagSet("testRedis-pubsub-pop-timeout", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "testRedis-pop-timeout-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		t.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		t.Skipf("Skipping TestRedisPopWithTimeout because Redis ping failed: %v", err)
	}

	topic := "testRedis-pop-timeout-channel"
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer subscription.Close(ctx)

	// Attempt to pop with timeout when no messages are published.
	_, err = subscription.Pop(ctx, 500*time.Millisecond)
	if err == nil {
		t.Errorf("Expected timeout error, but got none")
	}
}

// TestRedisBPopWithContextCancel ensures that BPop respects context cancellation.
func TestRedisBPopWithContextCancel(t *testing.T) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "testRedis-pubsub-context-cancel"
	flags := pflag.NewFlagSet("testRedis-pubsub-context-cancel", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "testRedis-context-cancel-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		t.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		t.Skipf("Skipping TestRedisBPopWithContextCancel because Redis ping failed: %v", err)
	}

	topic := "testRedis-context-cancel-channel"
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer subscription.Close(ctx)

	// Create a cancellable context.
	ctxCancel, cancel := context.WithCancel(ctx)

	// Cancel the context immediately.
	cancel()

	// Attempt to BPop.
	_, err = subscription.BPop(ctxCancel)
	if err == nil {
		t.Errorf("Expected context cancellation error, but got none")
	}
}

// BenchmarkRedisPublish measures the performance of publishing messages.
func BenchmarkRedisPublish(b *testing.B) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "benchmarkRedis-pubsub-publish"
	flags := pflag.NewFlagSet("benchmarkRedis-pubsub-publish", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "benchmarkRedis-publish-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		b.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		b.Skipf("Skipping BenchmarkRedisPublish because Redis ping failed: %v", err)
	}

	topic := "benchmarkRedis-publish-channel"

	// Start a goroutine to consume messages to prevent blocking
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		b.Fatalf("Failed to subscribe: %v", err)
	}
	defer subscription.Close(ctx)

	go func() {
		for {
			_, err := subscription.BPop(ctx)
			if err != nil {
				return
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		messageChannel := make(chan *TestMessage, 1)
		messageChannel <- &TestMessage{Content: fmt.Sprintf("BenchmarkMsg %d", i)}
		close(messageChannel)

		err := pubsub.Publish(ctx, topic, messageChannel, 1)
		if err != nil {
			b.Errorf("Publish failed: %v", err)
		}
	}
}

// BenchmarkRedisSubscribe measures the performance of subscribing and receiving messages.
func BenchmarkRedisSubscribe(b *testing.B) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "benchmarkRedis-pubsub-subscribe"
	flags := pflag.NewFlagSet("benchmarkRedis-pubsub-subscribe", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "benchmarkRedis-subscribe-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		b.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		b.Skipf("Skipping BenchmarkRedisSubscribe because Redis ping failed: %v", err)
	}

	topic := "benchmarkRedis-subscribe-channel"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		subscription, err := pubsub.Subscribe(ctx, topic)
		if err != nil {
			b.Fatalf("Failed to subscribe: %v", err)
		}

		// Publish a single message.
		messageChannel := make(chan *TestMessage, 1)
		messageChannel <- &TestMessage{Content: fmt.Sprintf("BenchmarkMsg %d", i)}
		close(messageChannel)

		go func() {
			pubsub.Publish(ctx, topic, messageChannel, 1)
		}()

		// Receive the message.
		_, err = subscription.BPop(ctx)
		if err != nil {
			b.Errorf("Failed to receive message: %v", err)
		}

		subscription.Close(ctx)
	}
}

// BenchmarkRedisPublishConcurrent measures the performance of concurrent publishing.
func BenchmarkRedisPublishConcurrent(b *testing.B) {
	ctx := context.Background()

	// Initialize flags and viper
	prefix := "benchmarkRedis-pubsub-publish-concurrent"
	flags := pflag.NewFlagSet("benchmarkRedis-pubsub-publish-concurrent", pflag.ContinueOnError)
	RedisPubSubFlags(prefix).AddFlagSet(flags)
	setupViper(prefix, flags, map[string]interface{}{
		"redis-address":   "localhost:6379",
		"default-channel": "benchmarkRedis-publish-concurrent-channel",
		"redis-password":  "",
		"redis-db":        0,
	})
	flags.Parse([]string{}) // No flags for testing

	// Create RedisPubSub client
	pubsub, err := NewRedisPubSubFromFlags[TestMessage](ctx, prefix)
	if err != nil {
		b.Fatalf("Failed to create RedisPubSub: %v", err)
	}
	defer pubsub.Close()

	// Ping the Redis server
	if err := pubsub.Ping(ctx, 2*time.Second); err != nil {
		b.Skipf("Skipping BenchmarkRedisPublishConcurrent because Redis ping failed: %v", err)
	}

	topic := "benchmarkRedis-publish-concurrent-channel"

	// Start a goroutine to consume messages to prevent blocking
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		b.Fatalf("Failed to subscribe: %v", err)
	}
	defer subscription.Close(ctx)

	go func() {
		for {
			_, err := subscription.BPop(ctx)
			if err != nil {
				return
			}
		}
	}()

	b.ResetTimer()

	var wg sync.WaitGroup
	concurrency := 10
	messagesPerGoroutine := b.N / concurrency

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				messageChannel := make(chan *TestMessage, 1)
				messageChannel <- &TestMessage{Content: fmt.Sprintf("ConcurrentMsg %d", start+j)}
				close(messageChannel)

				err := pubsub.Publish(ctx, topic, messageChannel, 1)
				if err != nil {
					b.Errorf("Publish failed: %v", err)
				}
			}
		}(i * messagesPerGoroutine)
	}

	wg.Wait()
}
