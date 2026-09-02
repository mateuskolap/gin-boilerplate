package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"gin-boilerplate/internal/domain"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles"`
}

func GenerateRandomToken(bytesLen int) (string, error) {
	bytes := make([]byte, bytesLen)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func HashSHA256(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func GenerateAccessToken(userID uuid.UUID, roles []string, secret string, expiration time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := CustomClaims{
		Subject:   userID.String(),
		ID:        uuid.New().String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
		NotBefore: jwt.NewNumericDate(now),
		Roles:     roles,
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := jwtToken.SignedString([]byte(secret))
	if err != nil {
		return "", domain.NewAppError(
			domain.ErrTypeInternal,
			"Error signing JWT token",
			err,
		)
	}

	return tokenString, nil
}

// ParseAndValidateJWT parses and validates a signed JWT token string, verifying its signing method and claims.
func ParseAndValidateJWT(tokenString string, secret string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
