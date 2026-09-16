package middleware

import (
	"fmt"
	"strconv"
	"time"

	"gin-boilerplate/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter limits requests per client IP on the current route using Redis INCR.
func RateLimiter(rdb *redis.Client, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		key := fmt.Sprintf("rate_limit:%s:%s", c.FullPath(), c.ClientIP())
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		if count > limit {
			retryAfter := int(window.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			_ = c.Error(domain.NewAppError(
				domain.ErrTypeTooManyRequests,
				"Too many requests. Please try again later.",
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}
