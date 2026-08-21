package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
)

type PermissionName string

const (
	PermissionCreateRole PermissionName = "create_role"
	PermissionUpdateRole PermissionName = "update_role"
	PermissionDeleteRole PermissionName = "delete_role"
	PermissionViewRole   PermissionName = "view_role"
)

var AllPermissions = []PermissionName{
	PermissionCreateRole,
	PermissionUpdateRole,
	PermissionDeleteRole,
	PermissionViewRole,
}

type permissionUseCase struct {
	permissionRepo domain.PermissionRepository
}

func NewPermissionUseCase(permissionRepo domain.PermissionRepository) domain.PermissionUseCase {
	return &permissionUseCase{
		permissionRepo: permissionRepo,
	}
}

func (p *permissionUseCase) SeedPermissions(ctx context.Context) error {
	permissions := make([]domain.Permission, 0, len(AllPermissions))

	for _, perm := range AllPermissions {
		permissions = append(permissions, domain.Permission{
			Name: string(perm),
		})
	}

	return p.permissionRepo.UpsertByName(ctx, permissions)
}
