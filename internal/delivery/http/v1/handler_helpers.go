package v1

import (
	"gin-boilerplate/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func extractUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Unauthorized",
			nil,
		)
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.Nil, domain.NewAppError(
			domain.ErrTypeValidation,
			"Invalid user ID format",
			err,
		)
	}

	return userID, nil
}

func bindJSON[T any](c *gin.Context) (T, error) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, domain.NewAppError(
			domain.ErrTypeValidation,
			"Invalid request payload: "+err.Error(),
			err,
		)
	}
	return req, nil
}

func extractToken(c *gin.Context) (string, error) {
	tokenString, exists := c.Get("raw_token")
	if !exists {
		return "", domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Token not found in request context",
			nil,
		)
	}
	return tokenString.(string), nil
}
