package middleware

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"

	"github.com/gin-gonic/gin"
)

func RateLimiter(limiter port.RateLimiter, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rate_limit:%s:%s", c.FullPath(), c.ClientIP())
		result, err := limiter.Allow(c.Request.Context(), key, limit, window)
		if err != nil {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeUnavailable,
				"Rate limiting service is unavailable",
				err,
			))
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(result.ResetAfter).Unix(), 10))

		if !result.Allowed {
			retryAfter := int(math.Ceil(result.RetryAfter.Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			_ = c.Error(shared.NewAppError(
				shared.ErrTypeTooManyRequests,
				"Too many requests. Please try again later.",
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}
