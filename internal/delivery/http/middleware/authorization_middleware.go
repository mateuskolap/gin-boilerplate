package middleware

import (
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"uuid"

	"github.com/gin-gonic/gin"
)

func RequirePermission(permission domain.PermissionName, permissionChecker domain.PermissionCheckerUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get("user_id")
		if !exists {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeUnauthorized,
				"Authenticated user not found in context",
				nil,
			))
			c.Abort()
			return
		}

		userIDString, ok := value.(string)
		if !ok {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeUnauthorized,
				"Authenticated user is invalid",
				nil,
			))
			c.Abort()
			return
		}
		userID, err := uuid.Parse(userIDString)
		if err != nil {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeUnauthorized,
				"Authenticated user is invalid",
				err,
			))
			c.Abort()
			return
		}

		allowed, err := permissionChecker.HasPermission(c.Request.Context(), userID, permission)
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
