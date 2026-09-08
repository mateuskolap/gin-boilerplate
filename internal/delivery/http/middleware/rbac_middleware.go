package middleware

import (
	"gin-boilerplate/internal/domain"

	"github.com/gin-gonic/gin"
)

func RequirePermission(permission string, permissionChecker domain.PermissionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("user_roles")
		if !exists {
			_ = c.Error(domain.NewAppError(
				domain.ErrTypeForbidden,
				"User roles not found in context",
				nil,
			))
			c.Abort()
			return
		}

		roles, ok := val.([]string)
		if !ok || len(roles) == 0 {
			_ = c.Error(domain.NewAppError(
				domain.ErrTypeForbidden,
				"User roles are invalid or empty",
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}
