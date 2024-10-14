package ps

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Seann-Moser/go-serve/pkg/clientpkg"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/option"
	"time"
)

var _ PubSub[any] = &GCPPubSub[any]{}

// GCPPubSub implements the PubSub interface using Google Cloud Pub/Sub.
type GCPPubSub[T any] struct {
	client              *pubsub.Client
	defaultTopic        string
	defaultSubscription string
}

func GCPPubSubFlags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(clientpkg.GetFlagWithPrefix(prefix, "gcp-pub-sub"), pflag.ExitOnError)
	// Define flags with the given prefix
	fs.String(clientpkg.GetFlagWithPrefix(prefix, "project-id"), "", "GCP Project ID")
	fs.String(clientpkg.GetFlagWithPrefix(prefix, "credentials-file"), "", "Path to GCP service account credentials JSON file")
	fs.String(clientpkg.GetFlagWithPrefix(prefix, "default-topic"), "", "Default Pub/Sub topic name")
	fs.String(clientpkg.GetFlagWithPrefix(prefix, "default-subscription"), "", "Default Pub/Sub subscription name")

	return fs
}

// NewGCPPubSubFromFlags parses the flags and creates a new GCPPubSub instance.
func NewGCPPubSubFromFlags[T any](ctx context.Context, prefix string) (*GCPPubSub[T], error) {
	// Retrieve flag values using clientpkg.GetFlagWithPrefix
	projectID := viper.GetString(clientpkg.GetFlagWithPrefix(prefix, "project-id"))
	credentialsFile := viper.GetString(clientpkg.GetFlagWithPrefix(prefix, "credentials-file"))
	defaultTopic := viper.GetString(clientpkg.GetFlagWithPrefix(prefix, "default-topic"))
	defaultSubscription := viper.GetString(clientpkg.GetFlagWithPrefix(prefix, "default-subscription"))

	// Validate required flags
	if projectID == "" {
		return nil, fmt.Errorf("project-id is required")
	}

	// Set up client options
	var clientOpts []option.ClientOption
	if credentialsFile != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(credentialsFile))
	}

	// Create the GCPPubSub client
	pubsubClient, err := NewGCPPubSub[T](ctx, projectID, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCPPubSub client: %w", err)
	}

	// Assign default topic and subscription if provided
	pubsubClient.defaultTopic = defaultTopic
	pubsubClient.defaultSubscription = defaultSubscription

	return pubsubClient, nil
}

// NewGCPPubSub creates a new GCPPubSub client.
func NewGCPPubSub[T any](ctx context.Context, projectID string, opts ...option.ClientOption) (*GCPPubSub[T], error) {
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}
	return &GCPPubSub[T]{
		client: client,
	}, nil
}

// Publish publishes a message to the specified topic.
func (g *GCPPubSub[T]) Publish(ctx context.Context, topic string, data chan *T, workers int) error {
	if topic == "" {
		topic = g.defaultTopic
	}
	t := g.client.Topic(topic)
	wg, ctx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		select {
		case <-ctx.Done():
			close(data)
			return ctx.Err()
		}
	})
	if workers <= 0 {
		workers = 1
	}
	ctxLogger.Info(ctx, "starting publisher with workers", zap.Int("workers", workers), zap.String("topic", topic))
	for i := 0; i < workers; i++ {
		wg.Go(func() error {
			for d := range data {
				ctxLogger.Info(ctx, "attempting to publish message")
				b, err := json.Marshal(d)
				if err != nil {
					ctxLogger.Error(ctx, "failed marshalling data", zap.Error(err))
					continue
				}
				result := t.Publish(ctx, &pubsub.Message{
					Data: b,
				})
				_, err = result.Get(ctx)
				if err != nil {
					ctxLogger.Warn(ctx, "failed to publish msg", zap.Error(err))
					continue
				}
			}
			ctxLogger.Info(ctx, "publisher worker finished")
			return nil
		})
	}
	go func() {
		err := wg.Wait()
		if err != nil {
			ctxLogger.Warn(ctx, "failed to publish msg", zap.Error(err))
		}
		ctxLogger.Info(ctx, "publisher worker finished")
	}()

	return nil
}

// CreateTopic creates a new Pub/Sub topic.
func (g *GCPPubSub[T]) CreateTopic(ctx context.Context, topic string) (*pubsub.Topic, error) {
	return g.client.CreateTopic(ctx, topic)
}

// Subscribe subscribes to a subscription and processes messages using the handler function.
func (g *GCPPubSub[T]) Subscribe(ctx context.Context, subscriptionName string) (*Subscription[T], error) {
	if subscriptionName == "" {
		subscriptionName = g.defaultSubscription
	}
	sub := g.client.Subscription(subscriptionName)
	subscription := &Subscription[T]{
		Name: subscriptionName,
		c:    make(chan *SubscriptionData[T]),
	}
	go func() {
		ctxLogger.Info(ctx, "starting subscripber", zap.String("subscription", subscriptionName))
		// Receive will start multiple goroutines to receive messages.
		err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			var d T
			ctxLogger.Info(ctx, "recieved message")
			err := json.Unmarshal(msg.Data, &d)
			if err != nil {
				msg.Nack()
				ctxLogger.Warn(ctx, "failed unmarshalling data")
				return
			}

			subscription.c <- &SubscriptionData[T]{
				data: &d,
				Ack: func(ctx context.Context) error {
					msg.Ack()
					return nil
				},
				Nack: func(ctx context.Context) error {
					msg.Nack()
					return nil
				},
			}
		})
		if err != nil {
			ctxLogger.Warn(ctx, "failed to receive subscription data", zap.Error(err))
		}
		ctxLogger.Info(ctx, "subscriber finished", zap.String("subscription", subscriptionName))
	}()

	return subscription, nil
}

// Close closes the Pub/Sub client.
// It should be called when the client is no longer needed.
func (g *GCPPubSub[T]) Close() error {
	return g.client.Close()
}

func (g *GCPPubSub[T]) Ping(ctx context.Context, timeout time.Duration) error {
	return nil
}
