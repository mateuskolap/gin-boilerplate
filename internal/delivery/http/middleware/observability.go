package middleware

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"log/slog"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, err := uuid.Parse(c.GetHeader("X-Request-ID"))
		if err != nil {
			requestID = uuid.New()
		}
		requestIDValue := requestID.String()

		c.Request = c.Request.WithContext(shared.WithRequestID(c.Request.Context(), requestID))
		c.Header("X-Request-ID", requestIDValue)
		c.Next()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		slog.InfoContext(c.Request.Context(), "http request",
			"request_id", requestIDFromContext(c.Request.Context()),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", c.ClientIP(),
			"bytes", c.Writer.Size(),
		)
	}
}

func requestIDFromContext(ctx context.Context) string {
	requestID := shared.RequestIDFromContext(ctx)
	if requestID == nil {
		return ""
	}
	return requestID.String()
}
