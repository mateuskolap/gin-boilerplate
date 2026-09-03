package dto

import (
	"gin-boilerplate/internal/domain"
	"uuid"
)

type PermissionResponse struct {
	ID   uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"`
	Name string    `json:"name" example:"create_user"`
}

func ToPermissionResponse(permission *domain.Permission) *PermissionResponse {
	return &PermissionResponse{
		ID:   permission.ID,
		Name: permission.Name,
	}
}
