package middleware

import (
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthenticationMiddleware(authUseCase domain.AuthUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeUnauthorized,
				"Authorization header is required",
				nil,
			))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeUnauthorized,
				"Authorization format must be Bearer <token>",
				nil,
			))
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := authUseCase.ValidateAccessToken(c.Request.Context(), tokenString)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}

		c.Set("user_id", claims.Subject)
		c.Set("raw_token", tokenString)
		c.Next()
	}
}
