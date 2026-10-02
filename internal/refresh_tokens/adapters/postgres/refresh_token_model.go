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

func refreshTokenModelFromDomain(entity *refreshtokendomain.RefreshToken) *RefreshTokenModel {
	model := &RefreshTokenModel{
		UserID: entity.UserID, TokenHash: entity.TokenHash, ExpiresAt: entity.ExpiresAt,
		RevokedAt: entity.RevokedAt, ReplacedBy: entity.ReplacedBy, IPAddress: entity.IpAddress, UserAgent: entity.UserAgent,
	}
	model.BaseModel = postgres.BaseModelFromDomain(entity.BaseModel)
	return model
}

func refreshTokenDomainFromModel(model *RefreshTokenModel) *refreshtokendomain.RefreshToken {
	entity := &refreshtokendomain.RefreshToken{
		UserID: model.UserID, TokenHash: model.TokenHash, ExpiresAt: model.ExpiresAt,
		RevokedAt: model.RevokedAt, ReplacedBy: model.ReplacedBy, IpAddress: model.IPAddress, UserAgent: model.UserAgent,
	}
	entity.BaseModel = model.BaseModel.ToDomain()
	return entity
}
