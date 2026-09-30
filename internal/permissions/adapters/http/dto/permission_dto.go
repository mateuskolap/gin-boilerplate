package dto

import (
	"gin-boilerplate/internal/delivery/http/dto"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
	"uuid"
)

type PermissionResponse struct {
	ID   uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"`
	Name string    `json:"name" example:"create_user"`
}

type PaginatedPermissionResponse = dto.PaginatedResponse[PermissionResponse]

func ToPermissionResponse(permission *permissiondomain.Permission) PermissionResponse {
	return PermissionResponse{
		ID:   permission.ID,
		Name: permission.Name,
	}
}
