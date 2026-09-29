package middleware

import (
	"net/http"
	"slices"

	"gin-boilerplate/internal/domain/shared"

	"github.com/gin-gonic/gin"
)

// ValidateOrigin protects cookie authentication against browser CSRF requests.
func ValidateOrigin(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		if origin == "" || origin == "null" || !slices.Contains(allowedOrigins, origin) {
			_ = c.Error(shared.NewAppError(shared.ErrTypeForbidden, "Request origin is not allowed", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}
