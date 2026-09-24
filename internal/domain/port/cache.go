package port

import (
	"context"
	"time"
)

type CacheRepository interface {
	// Set stores value under key with the requested time-to-live.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// Get loads the value for key into dest.
	Get(ctx context.Context, key string, dest any) error
}
