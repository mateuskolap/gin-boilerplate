package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
)

var allowedPermissionFilterFields = map[string]bool{
	"name":       true,
	"created_at": true,
	"id":         true,
}

type permissionUseCase struct {
	domain.BaseListUseCase[domain.Permission]
	permissionRepo domain.PermissionRepository
}

func NewPermissionUseCase(permissionRepo domain.PermissionRepository) domain.PermissionUseCase {
	return &permissionUseCase{
		BaseListUseCase: NewBaseListUseCase(
			permissionRepo,
			allowedPermissionFilterFields,
		),
		permissionRepo: permissionRepo,
	}
}

func (p *permissionUseCase) SeedPermissions(ctx context.Context) error {
	permissions := make([]domain.Permission, 0, len(domain.AllPermissions))

	for _, perm := range domain.AllPermissions {
		permissions = append(permissions, domain.Permission{
			Name: string(perm),
		})
	}

	return p.permissionRepo.UpsertByName(ctx, permissions)
}
