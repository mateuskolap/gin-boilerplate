package application

import (
	sharedapp "gin-boilerplate/internal/application"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
)

var allowedPermissionFilterFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"id":         true,
}

func NewPermissionUseCase(
	permissionRepo permissiondomain.PermissionRepository,
) permissiondomain.PermissionUseCase {
	return sharedapp.NewBaseListUseCase(permissionRepo, allowedPermissionFilterFields)
}
