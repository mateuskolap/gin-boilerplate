package domain

import (
	"context"
	"time"
	"uuid"
)

type RefreshToken struct {
	BaseModel
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
	BaseRepository[RefreshToken]
	FindByTokenHash(ctx context.Context, token string) (*RefreshToken, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, params PaginationParams, filters []Filter) (*PaginatedResult[RefreshToken], error)
}

type RefreshTokenUseCase interface {
	Create(ctx context.Context, userID uuid.UUID, ipAddress string, userAgent string) (string, error)
	Rotate(ctx context.Context, oldToken, ipAddress, userAgent string) (string, error)
	FindByTokenHash(ctx context.Context, token string) (*RefreshToken, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, params PaginationParams, filters []Filter) (*PaginatedResult[RefreshToken], error)
	Revoke(ctx context.Context, token string) error
	RevokeEntity(ctx context.Context, refreshToken *RefreshToken) error
	Validate(ctx context.Context, token string) (*RefreshToken, error)
}
