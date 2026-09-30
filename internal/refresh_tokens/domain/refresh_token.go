package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"time"
	"uuid"
)

type RefreshToken struct {
	shared.BaseModel
	UserID     uuid.UUID  `json:"user_id"`
	TokenHash  string     `json:"token"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	ReplacedBy *uuid.UUID `json:"replaced_by,omitempty"`
	IpAddress  string     `json:"ip_address"`
	UserAgent  string     `json:"user_agent"`
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	// FindByTokenHash retrieves a refresh token by its stored token hash.
	FindByTokenHash(ctx context.Context, token string) (*RefreshToken, error)
	// ListByUserID returns a paginated list of a user's refresh tokens matching filters.
	ListByUserID(ctx context.Context, userID uuid.UUID, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[RefreshToken], error)
	// RevokeAllByUserID revokes every refresh token associated with the user.
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error

	// Revoke atomically revokes an active refresh token owned by userID.
	// replacedBy is set when a refresh token is rotated.
	Revoke(ctx context.Context, id, userID uuid.UUID, replacedBy *uuid.UUID) (revoked bool, err error)
	// RevokeAllExcept revokes all active refresh tokens for a user except one.
	RevokeAllExcept(ctx context.Context, userID, exceptID uuid.UUID) error

	// DeleteExpiredBefore permanently removes refresh tokens expired before cutoff.
	DeleteExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type RefreshTokenUseCase interface {
	// Create generates a refresh token for the user and returns its plaintext value.
	Create(ctx context.Context, userID uuid.UUID, ipAddress string, userAgent string) (string, error)
	// Rotate validates oldToken, revokes it, and returns a replacement token.
	Rotate(ctx context.Context, oldToken, ipAddress, userAgent string) (string, error)
	// FindByTokenHash looks up a refresh token using its plaintext token value.
	FindByTokenHash(ctx context.Context, token string) (*RefreshToken, error)
	// ListActiveByUserID returns a paginated list of the user's unrevoked, unexpired tokens.
	ListActiveByUserID(ctx context.Context, userID uuid.UUID, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[RefreshToken], error)
	// Revoke revokes the refresh token identified by its plaintext value.
	Revoke(ctx context.Context, token string) error
	// RevokeAllByUserID revokes all refresh tokens belonging to the user.
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
	// RevokeSession revokes one active session owned by userID.
	RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error
	// RevokeOtherSessions revokes every active session except the session identified by currentRefreshToken.
	RevokeOtherSessions(ctx context.Context, userID uuid.UUID, currentRefreshToken string) error
	// Validate checks that token exists, is unrevoked, and has not expired.
	Validate(ctx context.Context, token string) (*RefreshToken, error)
}

// RefreshTokenMaintenanceUseCase contains maintenance operations that are run
// outside HTTP request flows.
type RefreshTokenMaintenanceUseCase interface {
	PurgeExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
