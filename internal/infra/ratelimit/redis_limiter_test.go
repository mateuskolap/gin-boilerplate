package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisLimiterPropagatesCanceledRequest(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0})
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewRedisLimiter(client).Allow(ctx, "key", 10, time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v, want canceled context", err)
	}
}
