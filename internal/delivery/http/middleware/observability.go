package middleware

import (
	"gin-boilerplate/internal/domain/shared"
	"log/slog"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, err := uuid.Parse(c.GetHeader("X-Request-ID"))
		if err != nil {
			requestID = uuid.New()
		}
		requestIDValue := requestID.String()

		c.Set(requestIDKey, requestIDValue)
		c.Header("X-Request-ID", requestIDValue)
		c.Next()
	}
}

func RequestIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(shared.WithRequestIP(c.Request.Context(), c.ClientIP()))
		c.Next()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		slog.InfoContext(c.Request.Context(), "http request",
			"request_id", requestIDFromContext(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", c.ClientIP(),
			"bytes", c.Writer.Size(),
		)
	}
}

func requestIDFromContext(c *gin.Context) string {
	requestID, _ := c.Get(requestIDKey)
	value, _ := requestID.(string)
	return value
}
