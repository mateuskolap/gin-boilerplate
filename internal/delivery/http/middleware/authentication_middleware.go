package middleware

import (
	"gin-boilerplate/internal/delivery/http/authcookie"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"strings"

	"uuid"

	"github.com/gin-gonic/gin"
)

func AuthenticationMiddleware(authUseCase domain.AuthUseCase, useCookies bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		if useCookies {
			tokenString = authcookie.Get(c, authcookie.AccessTokenName)
		} else {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				_ = c.Error(shared.NewAppError(shared.ErrTypeUnauthorized, "Authorization header is required", nil))
				c.Abort()
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				_ = c.Error(shared.NewAppError(shared.ErrTypeUnauthorized, "Authorization format must be Bearer <token>", nil))
				c.Abort()
				return
			}
			tokenString = parts[1]
		}

		if tokenString == "" {
			_ = c.Error(shared.NewAppError(shared.ErrTypeUnauthorized, "Access token is required", nil))
			c.Abort()
			return
		}

		claims, err := authUseCase.ValidateAccessToken(c.Request.Context(), tokenString)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			_ = c.Error(shared.NewAppError(shared.ErrTypeUnauthorized, "Invalid authenticated user", err))
			c.Abort()
			return
		}

		c.Set("user_id", claims.Subject)
		c.Set("raw_token", tokenString)
		c.Request = c.Request.WithContext(shared.WithActorID(c.Request.Context(), userID))
		c.Next()
	}
}
