package domain

import (
	"context"
	"time"
)

type TokenBlackList interface {
	// RevokeToken marks a token ID (jti) as revoked with a time-to-live matching token expiration.
	RevokeToken(ctx context.Context, jti string, expiresIn time.Duration) error

	// IsRevoked checks if a token ID (jti) is present in the blacklist.
	IsRevoked(ctx context.Context, jti string) (bool, error)
}
