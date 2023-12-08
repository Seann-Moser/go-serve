package clientpkg

import (
	"context"
	"github.com/Seann-Moser/go-serve/pkg/ctxLogger"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"strings"
	"time"
)

type BackOff struct {
	maxRetry        uint64
	maxInterval     time.Duration
	maxElapsedTime  time.Duration
	initialInterval time.Duration
}

func BackOffFlags(prefix string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(GetFlagWithPrefix("backoff", prefix), pflag.ExitOnError)
	fs.Uint64(GetFlagWithPrefix("max-retry", prefix), 5, strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "max-retry"))))
	fs.Duration(GetFlagWithPrefix("max-interval", prefix), 15*time.Second, strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "max-interval"))))
	fs.Duration(GetFlagWithPrefix("max-elapsed-time", prefix), 45*time.Second, strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "max-elapsed-time"))))
	fs.Duration(GetFlagWithPrefix("max-initial-interval", prefix), 100*time.Millisecond, strings.ToUpper(ToSnakeCase(GetFlagWithPrefix(prefix, "max-initial-interval"))))
	return fs
}

func NewBackoffFromFlags(prefix string) *BackOff {
	return &BackOff{
		maxRetry:        viper.GetUint64(GetFlagWithPrefix("max-retry", prefix)),
		maxInterval:     viper.GetDuration(GetFlagWithPrefix("max-interval", prefix)),
		maxElapsedTime:  viper.GetDuration(GetFlagWithPrefix("max-elapsed-time", prefix)),
		initialInterval: viper.GetDuration(GetFlagWithPrefix("max-initial-interval", prefix)),
	}
}

func NewBackoff(maxRetry uint64, maxInterval, maxElapsedTime, initialInterval time.Duration) *BackOff {
	return &BackOff{
		maxRetry:        maxRetry,
		maxInterval:     maxInterval,
		maxElapsedTime:  maxElapsedTime,
		initialInterval: initialInterval,
	}
}

func (b *BackOff) Retry(ctx context.Context, operation backoff.Operation) error {
	op := backoff.Operation(operation)
	notify := func(err error, backoffDuration time.Duration) {
		ctxLogger.Debug(ctx, "retrying", zap.Error(err), zap.Duration("backoff_duration", backoffDuration))
	}
	if err := backoff.RetryNotify(op, b.getBackoff(), notify); err != nil {
		return err
	}
	return nil

}
func (b *BackOff) getBackoff() backoff.BackOff {
	requestExpBackOff := backoff.NewExponentialBackOff()
	requestExpBackOff.InitialInterval = b.initialInterval
	requestExpBackOff.RandomizationFactor = 0.5
	requestExpBackOff.Multiplier = 1.5
	requestExpBackOff.MaxInterval = b.maxInterval
	requestExpBackOff.MaxElapsedTime = b.maxElapsedTime
	return backoff.WithMaxRetries(requestExpBackOff, b.maxRetry)
}
