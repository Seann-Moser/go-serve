// inmemory_pubsub_test.go
package ps

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Define a sample message type for testing.
type TestMessage struct {
	Content string
}

// TestPublishAndSubscribe verifies that messages published to a topic are received by subscribers.
func TestPublishAndSubscribe(t *testing.T) {
	ctx := context.Background()
	pubsub := NewInMemoryPubSub[TestMessage]()
	defer pubsub.Close()

	topic := "test-topic"
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer subscription.Close(ctx)

	messageChannel := make(chan *TestMessage)

	go func() {
		// Simulate a publisher.
		for i := 1; i <= 5; i++ {
			messageChannel <- &TestMessage{Content: fmt.Sprintf("Message %d", i)}
		}
		close(messageChannel)
	}()

	go func() {
		// Publish messages.
		err := pubsub.Publish(ctx, topic, messageChannel, 1)
		if err != nil {
			t.Errorf("Publish failed: %v", err)
		}
	}()

	// Collect received messages.
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

// TestMultipleSubscribers ensures that multiple subscribers receive all published messages.
func TestMultipleSubscribers(t *testing.T) {
	ctx := context.Background()
	pubsub := NewInMemoryPubSub[TestMessage]()
	defer pubsub.Close()

	topic := "multi-subscriber-topic"
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

	go func() {
		// Publish messages.
		for i := 1; i <= messageCount; i++ {
			messageChannel <- &TestMessage{Content: fmt.Sprintf("Msg %d", i)}
		}
		close(messageChannel)
	}()

	go func() {
		err := pubsub.Publish(ctx, topic, messageChannel, 1)
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

// TestPublishWithWorkers tests publishing with multiple worker goroutines.
func TestPublishWithWorkers(t *testing.T) {
	ctx := context.Background()
	pubsub := NewInMemoryPubSub[TestMessage]()
	defer pubsub.Close()

	topic := "worker-topic"
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer subscription.Close(ctx)

	messageCount := 100
	workers := 10

	messageChannel := make(chan *TestMessage)

	go func() {
		for i := 1; i <= messageCount; i++ {
			messageChannel <- &TestMessage{Content: fmt.Sprintf("WorkerMsg %d", i)}
		}
		close(messageChannel)
	}()

	go func() {
		err := pubsub.Publish(ctx, topic, messageChannel, workers)
		if err != nil {
			t.Errorf("Publish failed: %v", err)
		}
	}()

	received := make(map[string]bool)
	for i := 1; i <= messageCount; i++ {
		msg, err := subscription.BPop(ctx)
		if err != nil {
			t.Errorf("Failed to receive message: %v", err)
			break
		}
		received[msg.data.Content] = true
		msg.Ack(ctx)
	}

	// Verify all messages are received.
	if len(received) != messageCount {
		t.Errorf("Expected %d messages, received %d", messageCount, len(received))
	}
}

// TestSubscriptionClose checks that closing a subscription stops message delivery.
func TestSubscriptionClose(t *testing.T) {
	ctx := context.Background()
	pubsub := NewInMemoryPubSub[TestMessage]()
	defer pubsub.Close()

	topic := "close-subscription-topic"
	subscription, err := pubsub.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	messageChannel := make(chan *TestMessage)

	go func() {
		for i := 1; i <= 5; i++ {
			messageChannel <- &TestMessage{Content: fmt.Sprintf("CloseMsg %d", i)}
		}
		close(messageChannel)
	}()

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

// TestPubSubClose ensures that closing the Pub/Sub system prevents further publishing and subscriptions.
func TestPubSubClose(t *testing.T) {
	ctx := context.Background()
	pubsub := NewInMemoryPubSub[TestMessage]()

	topic := "close-pubsub-topic"

	// Close the PubSub system.
	err := pubsub.Close()
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

// TestPopWithTimeout validates the Pop method with a timeout.
func TestPopWithTimeout(t *testing.T) {
	ctx := context.Background()
	pubsub := NewInMemoryPubSub[TestMessage]()
	defer pubsub.Close()

	topic := "timeout-topic"
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

// TestBPopWithContextCancel ensures that BPop respects context cancellation.
func TestBPopWithContextCancel(t *testing.T) {
	pubsub := NewInMemoryPubSub[TestMessage]()
	defer pubsub.Close()

	topic := "context-cancel-topic"
	subscription, err := pubsub.Subscribe(context.Background(), topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer subscription.Close(context.Background())

	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately.
	cancel()

	// Attempt to BPop.
	_, err = subscription.BPop(ctx)
	if err == nil {
		t.Errorf("Expected context cancellation error, but got none")
	}
}
