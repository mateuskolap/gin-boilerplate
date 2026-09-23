package health

import (
	"context"
	"database/sql"
	"fmt"

	"gin-boilerplate/internal/domain/port"

	"github.com/redis/go-redis/v9"
)

type Checker struct {
	db    *sql.DB
	redis *redis.Client
}

func NewChecker(db *sql.DB, redisClient *redis.Client) port.HealthChecker {
	return &Checker{db: db, redis: redisClient}
}

func (c *Checker) Readiness(ctx context.Context) error {
	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database is unavailable: %w", err)
	}
	if err := c.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis is unavailable: %w", err)
	}
	return nil
}
