package domain

import (
	"context"
	"time"
)

type CacheRepository interface {
	// Set stores a value under the given key with an expiration duration.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Get retrieves a string value by key. Returns empty string and nil error if not found.
	Get(ctx context.Context, key string) (string, error)

	// Delete removes a key and its value from cache.
	Delete(ctx context.Context, key string) error
}
