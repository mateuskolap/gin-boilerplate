package dto

import (
	"gin-boilerplate/internal/domain"
	"time"

	"github.com/google/uuid"
)

type RoleResponse struct {
	ID        uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"`
	Name      string    `json:"name" example:"Admin"`
	CreatedAt time.Time `json:"created_at" example:"2026-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-01T00:00:00Z"`
}

type CreateRoleRequest struct {
	Name string `json:"name" binding:"required,min=2" example:"Admin"`
}

type UpdateRoleRequest struct {
	Name string `json:"name" binding:"required,min=2" example:"Admin Updated"`
}

func ToRoleResponse(role *domain.Role) RoleResponse {
	return RoleResponse{
		ID:        role.ID,
		Name:      role.Name,
		CreatedAt: role.CreatedAt,
		UpdatedAt: role.UpdatedAt,
	}
}
