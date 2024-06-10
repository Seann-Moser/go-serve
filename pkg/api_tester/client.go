package api_tester

import "time"

type Client struct {
	MaxRequests         int64
	MaxRequestPerMinute int64

}

type LatencyStats struct {
	PayloadSize int64
	Duration    time.Duration
	startTime   time.Time
	endTime     time.Time
}

type RequestBuilder interface {
	GetBody() ([]byte, error)
}
