package middle

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

// RequestTracker uses atomic operations to track in-flight requests.
type RequestTracker struct {
	inFlight atomic.Int64
	done     chan struct{}
}

// NewRequestTracker initializes a new RequestTracker.
func NewRequestTracker() *RequestTracker {
	return &RequestTracker{
		inFlight: atomic.Int64{},
		done:     make(chan struct{}, 1),
	}
}

// TrackMiddleware is the middleware that tracks the number of in-flight requests using atomic operations.
func (rt *RequestTracker) TrackMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Atomically increment the in-flight counter
		rt.inFlight.Add(1)
		// Ensure decrement happens after the request completes
		defer func() {
			rt.inFlight.Add(-1)
		}()

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}

func (rt *RequestTracker) Done(ctx context.Context) <-chan struct{} {
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				rt.done <- struct{}{}
				return
			case <-ticker.C:
				if rt.inFlight.Load() > 0 {
					continue
				}
				rt.done <- struct{}{}
				return
			}
		}
	}()
	return rt.done
}
