package port

import (
	"context"
	"time"
)

type RateLimitResult struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
	ResetAfter time.Duration
}

type RateLimiter interface {
	// Allow checks whether a request identified by key fits within limit requests
	// during window and returns the current limit and retry timing information.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error)
}
