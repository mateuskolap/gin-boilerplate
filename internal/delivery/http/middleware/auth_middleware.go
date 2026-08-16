package middleware

import (
	"errors"
	"gin-boilerplate/internal/domain"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret string, blacklist domain.TokenBlackList) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			HandleError(c, domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Authorization header is required",
				nil,
			))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			HandleError(c, domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Authorization format must be Bearer <token>",
				nil,
			))
			c.Abort()
			return
		}

		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			HandleError(c, domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Invalid or expired token",
				err,
			))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok {
			HandleError(c, domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Invalid token claims",
				nil,
			))
			c.Abort()
			return
		}

		isRevoked, err := blacklist.IsRevoked(c.Request.Context(), claims.ID)
		if err != nil {
			HandleError(c, domain.NewAppError(
				domain.ErrTypeInternal,
				"Failed to check token revocation status",
				err,
			))
			c.Abort()
			return
		}

		if isRevoked {
			HandleError(c, domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Token has been revoked",
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
