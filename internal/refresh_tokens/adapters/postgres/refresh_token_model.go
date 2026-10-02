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
	model := &RefreshTokenModel{
		UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt,
		RevokedAt: token.RevokedAt, ReplacedBy: token.ReplacedBy, IPAddress: token.IpAddress, UserAgent: token.UserAgent,
	}
	model.BaseModel = postgres.BaseModelFromDomain(token.BaseModel)
	return model
}

func refreshTokenDomainFromModel(m *RefreshTokenModel) *refreshtokendomain.RefreshToken {
	token := &refreshtokendomain.RefreshToken{
		UserID: m.UserID, TokenHash: m.TokenHash, ExpiresAt: m.ExpiresAt,
		RevokedAt: m.RevokedAt, ReplacedBy: m.ReplacedBy, IpAddress: m.IPAddress, UserAgent: m.UserAgent,
	}
	token.BaseModel = m.BaseModel.ToDomain()
	return token
}
