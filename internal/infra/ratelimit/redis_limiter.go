package ratelimit

import (
	"context"
	"time"

	"gin-boilerplate/internal/domain/port"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type RedisLimiter struct {
	limiter *redis_rate.Limiter
}

func NewRedisLimiter(client *redis.Client) port.RateLimiter {
	return &RedisLimiter{limiter: redis_rate.NewLimiter(client)}
}

func (l *RedisLimiter) Allow(
	ctx context.Context,
	key string,
	limit int,
	window time.Duration,
) (*port.RateLimitResult, error) {
	result, err := l.limiter.Allow(ctx, key, redis_rate.Limit{
		Rate:   limit,
		Burst:  limit,
		Period: window,
	})
	if err != nil {
		return nil, err
	}

	return &port.RateLimitResult{
		Allowed:    result.Allowed > 0,
		Limit:      limit,
		Remaining:  result.Remaining,
		RetryAfter: result.RetryAfter,
		ResetAfter: result.ResetAfter,
	}, nil
}
