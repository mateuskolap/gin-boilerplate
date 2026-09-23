package domain

import (
	"context"
	"time"
)

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
}

type TokenClaims struct {
	Subject string
	TokenID string
}

type TokenBlackListRepository interface {
	// RevokeToken marks a token ID (jti) as revoked with a time-to-live matching token expiration.
	RevokeToken(ctx context.Context, jti string, expiresIn time.Duration) error

	// IsRevoked checks if a token ID (jti) is present in the blacklist.
	IsRevoked(ctx context.Context, jti string) (bool, error)

	// RevokeUserTokens records the timestamp of revocation for all tokens of a user with a TTL.
	RevokeUserTokens(ctx context.Context, userID string, expiresIn time.Duration) error

	// IsUserTokenRevoked checks if a token was issued before the user's revocation timestamp.
	IsUserTokenRevoked(ctx context.Context, userID string, issuedAt time.Time) (bool, error)
}

type AuthUseCase interface {
	// Register validates, hashes credentials, and creates a new user account.
	Register(ctx context.Context, user *User) error

	// Login verifies credentials and generates access and refresh tokens.
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (*AuthTokens, error)

	// Refresh verifies user has a valid refresh token and generates a signed JWT token string.
	Refresh(ctx context.Context, refreshToken, ipAddress, userAgent string) (*AuthTokens, error)

	// Logout invalidates a JWT token by adding its ID to the blacklist and revoking refresh token.
	Logout(ctx context.Context, accessToken, refreshToken string) error

	// ValidateAccessToken verifies token signature, expiration, and checks the blacklist.
	ValidateAccessToken(ctx context.Context, tokenString string) (*TokenClaims, error)
}
