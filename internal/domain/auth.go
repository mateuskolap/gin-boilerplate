package domain

import (
	"context"
	"time"
)

type TokenBlackList interface {
	RevokeToken(ctx context.Context, jti string, expiresIn time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}
