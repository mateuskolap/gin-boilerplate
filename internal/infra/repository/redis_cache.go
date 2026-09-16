package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gin-boilerplate/internal/domain/port"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) port.CacheRepository {
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

func (r *redisCache) SetAdd(ctx context.Context, key string, members []string, ttl time.Duration) error {
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, key)

	if len(members) > 0 {
		items := make([]interface{}, len(members))
		for i, m := range members {
			items[i] = m
		}
		pipe.SAdd(ctx, key, items...)
	}

	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisCache) SetMembers(ctx context.Context, key string) ([]string, error) {
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, nil
	}

	return r.client.SMembers(ctx, key).Result()
}

func (r *redisCache) CheckSetMembers(ctx context.Context, keys []string, member string) (bool, []string, error) {
	if len(keys) == 0 {
		return false, nil, nil
	}

	type keyCmds struct {
		key      string
		exists   *redis.IntCmd
		isMember *redis.BoolCmd
	}

	cmds := make([]keyCmds, len(keys))
	pipe := r.client.Pipeline()

	for i, k := range keys {
		cmds[i] = keyCmds{
			key:      k,
			exists:   pipe.Exists(ctx, k),
			isMember: pipe.SIsMember(ctx, k, member),
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return false, nil, err
	}

	var missingKeys []string
	for _, cmd := range cmds {
		if cmd.isMember.Val() {
			return true, nil, nil
		}
		if cmd.exists.Val() == 0 {
			missingKeys = append(missingKeys, cmd.key)
		}
	}

	return false, missingKeys, nil
}

func (r *redisCache) DeleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys by pattern: %w", err)
		}

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("failed to delete keys by pattern: %w", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}
