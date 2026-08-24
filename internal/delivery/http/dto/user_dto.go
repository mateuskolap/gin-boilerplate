package dto

import (
	"gin-boilerplate/internal/domain"
	"time"

	"uuid"
)

type UserProfileResponse struct {
	ID        uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"`
	Name      string    `json:"name" example:"John Doe"`
	Email     string    `json:"email" example:"john.doe@example.com"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-01T00:00:00Z"`
}

type UpdateProfileRequest struct {
	Name string `json:"name" binding:"required,min=2" example:"John Doe Updated"`
}

type UpdateUserRolesRequest struct {
	RoleIDs []uuid.UUID `json:"role_ids" binding:"required,min=1,dive,required" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d,f9e8d7c6-b5a4-3210-fedc-ba0987654321"`
}

func ToUserProfileResponse(user *domain.User) UserProfileResponse {
	return UserProfileResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
