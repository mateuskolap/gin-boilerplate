package usecase

import (
	"gin-boilerplate/internal/domain"
)

var allowedPermissionFilterFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"id":         true,
}

func NewPermissionUseCase(
	permissionRepo domain.PermissionRepository,
) domain.PermissionUseCase {
	return NewBaseListUseCase(permissionRepo, allowedPermissionFilterFields)
}
