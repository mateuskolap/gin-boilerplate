package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
)

var allowedPermissionFilterFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"id":         true,
}

type permissionUseCase struct {
	shared.BaseListUseCase[domain.Permission]
	permissionRepo     domain.PermissionRepository
	rolePermissionRepo domain.RolePermissionRepository
}

func NewPermissionUseCase(
	permissionRepo domain.PermissionRepository,
	rolePermissionRepo domain.RolePermissionRepository,
) domain.PermissionUseCase {
	return &permissionUseCase{
		BaseListUseCase: NewBaseListUseCase(
			permissionRepo,
			allowedPermissionFilterFields,
		),
		permissionRepo:     permissionRepo,
		rolePermissionRepo: rolePermissionRepo,
	}
}

func (p *permissionUseCase) SeedPermissions(ctx context.Context) error {
	permissions := make([]domain.Permission, 0, len(domain.AllPermissions))

	for _, perm := range domain.AllPermissions {
		permissions = append(permissions, domain.Permission{
			Name: string(perm),
		})
	}

	if err := p.permissionRepo.UpsertByName(ctx, permissions); err != nil {
		return err
	}

	return p.rolePermissionRepo.InvalidateAll(ctx)
}
