package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gin-boilerplate/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) domain.CacheRepository {
	return &redisCache{
		client: client,
	}
}

func (r *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("erro ao serializar para o cache: %w", err)
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *redisCache) Get(ctx context.Context, key string, dest any) error {
	bytes, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return err
	}

	if err := json.Unmarshal(bytes, dest); err != nil {
		return fmt.Errorf("erro ao desserializar do cache: %w", err)
	}

	return nil
}

func (r *redisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
