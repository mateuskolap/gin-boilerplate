package dto

import (
	"gin-boilerplate/internal/domain"
	"time"
	"uuid"
)

type RefreshTokenResponse struct {
	ID        uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"`
	UserID    uuid.UUID `json:"user_id" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"`
	ExpiresAt time.Time `json:"expires_at" example:"2026-01-01T00:00:00Z"`
	IpAddress string    `json:"ip_address" example:"192.168.1.1"`
	UserAgent string    `json:"user_agent" example:"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-01T00:00:00Z"`
}

type PaginatedRefreshTokenResponse = PaginatedResponse[RefreshTokenResponse]

func ToRefreshTokenResponse(refreshToken *domain.RefreshToken) RefreshTokenResponse {
	return RefreshTokenResponse{
		ID:        refreshToken.ID,
		UserID:    refreshToken.UserID,
		ExpiresAt: refreshToken.ExpiresAt,
		IpAddress: refreshToken.IpAddress,
		UserAgent: refreshToken.UserAgent,
		CreatedAt: refreshToken.CreatedAt,
		UpdatedAt: refreshToken.UpdatedAt,
	}
}
