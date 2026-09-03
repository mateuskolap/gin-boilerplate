package dto

import (
	"gin-boilerplate/internal/domain"
	"time"

	"uuid"
)

type UserResponse struct {
	ID        uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"`
	Name      string    `json:"name" example:"John Doe"`
	Email     string    `json:"email" example:"john.doe@example.com"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-01T00:00:00Z"`
}

type UserWithRoleResponse struct {
	UserResponse
	Roles []RoleResponse `json:"roles"`
}

type PaginatedUserResponse = PaginatedResponse[UserResponse]

type UpdateProfileRequest struct {
	Name string `json:"name" binding:"required,min=2" example:"John Doe Updated"`
}

type UpdateUserRolesRequest struct {
	RoleIDs []uuid.UUID `json:"role_ids" binding:"required,min=1,dive,required" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d,f9e8d7c6-b5a4-3210-fedc-ba0987654321"`
}

func ToUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func ToUserWithRoleResponse(user *domain.User) UserWithRoleResponse {
	roles := make([]RoleResponse, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = ToRoleResponse(&role)
	}

	return UserWithRoleResponse{
		UserResponse: ToUserResponse(user),
		Roles:        roles,
	}
}
