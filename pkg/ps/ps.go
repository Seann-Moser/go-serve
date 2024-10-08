package ps

import (
	"cloud.google.com/go/pubsub"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// PubSub is the main interface that encompasses both Publisher and Subscriber functionalities.
type PubSub[T any] interface {
	Publisher[T]
	Subscriber[T]
}

// Publisher defines methods for publishing messages.
type Publisher[T any] interface {
	Publish(ctx context.Context, topic string, data chan *T, workers int) error
}

// Subscriber defines methods for subscribing to messages.
type Subscriber[T any] interface {
	Subscribe(ctx context.Context, subscription string) (*Subscription[T], error)
}

// MessageHandler is a callback function type for processing messages.
type MessageHandler func(ctx context.Context, msg *pubsub.Message)

type Subscription[T any] struct {
	Name      string
	c         chan *SubscriptionData[T]
	closeOnce sync.Once
	closeFunc func()
}

type SubscriptionData[T any] struct {
	data *T
	Ack  func(ctx context.Context) error
	Nack func(ctx context.Context) error
}

func (s *Subscription[T]) BPop(ctx context.Context) (*SubscriptionData[T], error) {
	select {
	case msg, ok := <-s.c:
		if !ok {
			return nil, errors.New("subscription closed")
		}
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Subscription[T]) Pop(ctx context.Context, timeout time.Duration) (*SubscriptionData[T], error) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-t.C:
		return nil, fmt.Errorf("timed out waiting for message")
	case msg := <-s.c:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Subscription[T]) Close(ctx context.Context) {
	if s.closeFunc != nil {
		s.closeFunc()
	} else {
		close(s.c)
	}
}
