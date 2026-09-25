package domain

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"time"
	"uuid"
)

type RefreshToken struct {
	shared.BaseModel
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TokenHash  string     `json:"token" gorm:"not null;uniqueIndex"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null;index"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty" gorm:"default:null"`
	ReplacedBy *uuid.UUID `json:"replaced_by,omitempty" gorm:"type:uuid;default:null"`
	IpAddress  string     `json:"ip_address" gorm:"not null"`
	UserAgent  string     `json:"user_agent" gorm:"not null"`

	User User `json:"user" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type RefreshTokenRepository interface {
	shared.BaseRepository[RefreshToken]
	// FindByTokenHash retrieves a refresh token by its stored token hash.
	FindByTokenHash(ctx context.Context, token string) (*RefreshToken, error)
	// ListByUserID returns a paginated list of a user's refresh tokens matching filters.
	ListByUserID(ctx context.Context, userID uuid.UUID, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[RefreshToken], error)
	// RevokeAllByUserID revokes every refresh token associated with the user.
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error

	// RevokeByID atomically revokes a refresh token by ID only if it has not been revoked yet.
	// Returns true if successfully revoked, or false if already revoked by a concurrent request.
	RevokeByID(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) (revoked bool, err error)

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
	// RevokeEntity marks the supplied refresh token as revoked if it is still active.
	RevokeEntity(ctx context.Context, refreshToken *RefreshToken) error
	// RevokeAllByUserID revokes all refresh tokens belonging to the user.
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
	// Validate checks that token exists, is unrevoked, and has not expired.
	Validate(ctx context.Context, token string) (*RefreshToken, error)
}

// RefreshTokenMaintenanceUseCase contains maintenance operations that are run
// outside HTTP request flows.
type RefreshTokenMaintenanceUseCase interface {
	PurgeExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
