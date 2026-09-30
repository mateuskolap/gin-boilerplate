package postgres

import (
	"time"

	"gin-boilerplate/internal/infra/postgres"
	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
	userpostgres "gin-boilerplate/internal/users/adapters/postgres"
	"uuid"
)

type RefreshTokenModel struct {
	postgres.BaseModel
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash  string    `gorm:"not null;uniqueIndex"`
	ExpiresAt  time.Time `gorm:"not null;index"`
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID             `gorm:"type:uuid"`
	IPAddress  string                 `gorm:"not null"`
	UserAgent  string                 `gorm:"not null"`
	User       userpostgres.UserModel `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (RefreshTokenModel) TableName() string { return "refresh_tokens" }

func refreshTokenModelFromDomain(token *refreshtokendomain.RefreshToken) *RefreshTokenModel {
	return &RefreshTokenModel{
		ID: token.ID, CreatedAt: token.CreatedAt, UpdatedAt: token.UpdatedAt,
		UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt,
		RevokedAt: token.RevokedAt, ReplacedBy: token.ReplacedBy, IPAddress: token.IpAddress, UserAgent: token.UserAgent,
	}
}

func refreshTokenDomainFromModel(model any) *refreshtokendomain.RefreshToken {
	m := model.(*RefreshTokenModel)
	token := &refreshtokendomain.RefreshToken{
		UserID: m.UserID, TokenHash: m.TokenHash, ExpiresAt: m.ExpiresAt,
		RevokedAt: m.RevokedAt, ReplacedBy: m.ReplacedBy, IpAddress: m.IPAddress, UserAgent: m.UserAgent,
	}
	token.ID, token.CreatedAt, token.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	return token
}
