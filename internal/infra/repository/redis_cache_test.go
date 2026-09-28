package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisCacheRejectsUnserializableValuesAndCanceledCalls(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewRedisCache(client)
	if err := cache.Set(context.Background(), "key", make(chan int), time.Minute); err == nil {
		t.Fatal("Set() accepted a value that cannot be serialized as JSON")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cache.Set(ctx, "key", "value", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("Set() canceled call error = %v", err)
	}
	var value string
	if err := cache.Get(ctx, "key", &value); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get() canceled call error = %v", err)
	}
}
