package middleware

import (
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"

	"github.com/gin-gonic/gin"
)

func RequirePermission(permission domain.PermissionName, permissionChecker domain.PermissionCheckerUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("user_roles")
		if !exists {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeForbidden,
				"User roles not found in context",
				nil,
			))
			c.Abort()
			return
		}

		roles, ok := val.([]string)
		if !ok || len(roles) == 0 {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeForbidden,
				"User roles are invalid or empty",
				nil,
			))
			c.Abort()
			return
		}

		allowed, err := permissionChecker.HasPermission(c.Request.Context(), roles, string(permission))
		if err != nil {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeInternal,
				"Failed to check permissions",
				err,
			))
			c.Abort()
			return
		}

		if !allowed {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeForbidden,
				"Insufficient permissions",
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}
