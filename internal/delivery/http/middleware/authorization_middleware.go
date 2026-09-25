package middleware

import (
	"errors"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"uuid"

	"github.com/gin-gonic/gin"
)

// OwnershipCheck determines whether the authenticated user owns the requested resource.
// It may inspect route parameters or load the resource through a use case.
type OwnershipCheck func(c *gin.Context, userID uuid.UUID) (bool, error)

func RequirePermission(permission domain.PermissionName, permissionChecker domain.PermissionCheckerUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := authenticatedUserID(c)
		if !ok || !assertUserHasPermission(c, userID, permission, permissionChecker) {
			return
		}

		c.Next()
	}
}

// RequirePermissionOrOwner allows the resource owner or a user with the permission.
func RequirePermissionOrOwner(permission domain.PermissionName, permissionChecker domain.PermissionCheckerUseCase, checkOwnership OwnershipCheck) gin.HandlerFunc {
	if checkOwnership == nil {
		panic("ownership check is required")
	}

	return func(c *gin.Context) {
		userID, ok := authenticatedUserID(c)
		if !ok {
			return
		}

		isOwner, err := checkOwnership(c, userID)
		if err != nil {
			var appErr *shared.AppError
			if !errors.As(err, &appErr) {
				err = shared.NewAppError(shared.ErrTypeInternal, "Failed to check resource ownership", err)
			}
			_ = c.Error(err)
			c.Abort()
			return
		}

		if !isOwner && !assertUserHasPermission(c, userID, permission, permissionChecker) {
			return
		}

		c.Next()
	}
}

// OwnerFromUserIDParam checks a route parameter that directly identifies the owning user.
func OwnerFromUserIDParam(paramName string) OwnershipCheck {
	return func(c *gin.Context, userID uuid.UUID) (bool, error) {
		ownerID, err := uuid.Parse(c.Param(paramName))
		if err != nil {
			return false, shared.NewAppError(shared.ErrTypeValidation, "Invalid ID format", err)
		}
		return ownerID == userID, nil
	}
}

func authenticatedUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		_ = c.Error(shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Authenticated user not found in context",
			nil,
		))
		c.Abort()
		return uuid.Nil(), false
	}

	userIDString, ok := value.(string)
	if !ok {
		_ = c.Error(shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Authenticated user is invalid",
			nil,
		))
		c.Abort()
		return uuid.Nil(), false
	}
	userID, err := uuid.Parse(userIDString)
	if err != nil {
		_ = c.Error(shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Authenticated user is invalid",
			err,
		))
		c.Abort()
		return uuid.Nil(), false
	}

	return userID, true
}

func assertUserHasPermission(c *gin.Context, userID uuid.UUID, permission domain.PermissionName, permissionChecker domain.PermissionCheckerUseCase) bool {
	allowed, err := permissionChecker.HasPermission(c.Request.Context(), userID, permission)
	if err != nil {
		_ = c.Error(shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to check permissions",
			err,
		))
		c.Abort()
		return false
	}

	if !allowed {
		_ = c.Error(shared.NewAppError(
			shared.ErrTypeForbidden,
			"Insufficient permissions",
			nil,
		))
		c.Abort()
		return false
	}

	return true
}
