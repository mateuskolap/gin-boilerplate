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
	Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error)
}
