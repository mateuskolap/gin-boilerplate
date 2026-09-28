package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"

	"gin-boilerplate/internal/delivery/http/response"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		slog.ErrorContext(c.Request.Context(), "request panic",
			"request_id", requestIDFromContext(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"error", fmt.Sprint(recovered),
			"stack", string(debug.Stack()),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.ApiResponse{
			Success: false,
			Error:   "Internal server error",
		})
	})
}
