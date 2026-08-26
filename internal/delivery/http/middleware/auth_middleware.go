package middleware

import (
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/pkg/security"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtSecret string, blacklist domain.TokenBlackList) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			_ = c.Error(domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Authorization header is required",
				nil,
			))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			_ = c.Error(domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Authorization format must be Bearer <token>",
				nil,
			))
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := security.ParseAndValidateJWT(tokenString, jwtSecret)
		if err != nil {
			_ = c.Error(domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Invalid or expired token",
				err,
			))
			c.Abort()
			return
		}

		isRevoked, err := blacklist.IsRevoked(c.Request.Context(), claims.ID)
		if err != nil {
			_ = c.Error(domain.NewAppError(
				domain.ErrTypeInternal,
				"Failed to check token revocation status",
				err,
			))
			c.Abort()
			return
		}

		if isRevoked {
			_ = c.Error(domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Invalid or expired token",
				nil,
			))
			c.Abort()
			return
		}

		c.Set("user_id", claims.Subject)
		c.Set("raw_token", tokenString)
		c.Next()
	}
}
