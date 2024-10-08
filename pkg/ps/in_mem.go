package ps

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// InMemoryPubSub is an in-memory implementation of the PubSub interface.
// It is suitable for testing or scenarios where external dependencies are not desired.
type InMemoryPubSub[T any] struct {
	mu          sync.RWMutex
	dispatchMu  sync.Mutex
	subscribers map[string][]chan *SubscriptionData[T]
	closed      bool
}

// NewInMemoryPubSub creates a new instance of InMemoryPubSub.
func NewInMemoryPubSub[T any]() *InMemoryPubSub[T] {
	return &InMemoryPubSub[T]{
		subscribers: make(map[string][]chan *SubscriptionData[T]),
	}
}

// Publish publishes messages to the specified topic.
func (im *InMemoryPubSub[T]) Publish(ctx context.Context, topic string, data chan *T, workers int) error {
	im.mu.RLock()
	if im.closed {
		im.mu.RUnlock()
		return fmt.Errorf("pubsub is closed")
	}
	im.mu.RUnlock()

	// Create worker goroutines to process messages.
	var wg sync.WaitGroup
	workerCount := workers
	if workerCount <= 0 {
		workerCount = 1
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range data {
				// Marshal and unmarshal to create a deep copy.
				b, err := json.Marshal(msg)
				if err != nil {
					// In a real implementation, handle the error appropriately.
					continue
				}

				var dataDecoded T
				err = json.Unmarshal(b, &dataDecoded)
				if err != nil {
					continue
				}

				// Create SubscriptionData
				subData := &SubscriptionData[T]{
					data: &dataDecoded,
					Ack: func(ctx context.Context) error {
						// Ack is a no-op in in-memory implementation.
						return nil
					},
					Nack: func(ctx context.Context) error {
						// Nack is a no-op in in-memory implementation.
						return nil
					},
				}

				// Lock dispatchMu to prevent concurrent send and closure
				im.dispatchMu.Lock()
				im.mu.RLock()
				currentSubs, exists := im.subscribers[topic]
				im.mu.RUnlock()
				if exists {
					for _, subCh := range currentSubs {
						select {
						case subCh <- subData:
						case <-ctx.Done():
							im.dispatchMu.Unlock()
							return
						}
					}
				}
				im.dispatchMu.Unlock()
			}
		}()
	}

	wg.Wait()
	return nil
}

// Subscribe subscribes to a given subscription (topic) and returns a Subscription.
func (im *InMemoryPubSub[T]) Subscribe(ctx context.Context, subscription string) (*Subscription[T], error) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.closed {
		return nil, fmt.Errorf("pubsub is closed")
	}

	// Create a channel for the subscription.
	subCh := make(chan *SubscriptionData[T], 100) // Buffered channel to prevent blocking.

	// Add the subscription channel to the subscribers map.
	im.subscribers[subscription] = append(im.subscribers[subscription], subCh)

	subscriptionObj := &Subscription[T]{
		Name: subscription,
		c:    subCh,
	}

	// Initialize closeOnce and define closeFunc using sync.Once.
	subscriptionObj.closeOnce = sync.Once{}
	subscriptionObj.closeFunc = func() {
		subscriptionObj.closeOnce.Do(func() {
			im.dispatchMu.Lock()
			defer im.dispatchMu.Unlock()

			im.mu.Lock()
			defer im.mu.Unlock()
			// Remove the subscription channel from the subscribers map.
			subs := im.subscribers[subscription]
			for i, ch := range subs {
				if ch == subCh {
					im.subscribers[subscription] = append(subs[:i], subs[i+1:]...)
					close(ch) // Safe to close now.
					break
				}
			}
			// If no more subscribers for the topic, delete the entry.
			if len(im.subscribers[subscription]) == 0 {
				delete(im.subscribers, subscription)
			}
		})
	}

	return subscriptionObj, nil
}

// Close shuts down the InMemoryPubSub and closes all subscription channels.
func (im *InMemoryPubSub[T]) Close() error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.closed {
		return fmt.Errorf("pubsub is already closed")
	}

	im.closed = true
	im.dispatchMu.Lock()
	defer im.dispatchMu.Unlock()
	for topic, subs := range im.subscribers {
		for _, subCh := range subs {
			close(subCh)
		}
		delete(im.subscribers, topic)
	}

	return nil
}
