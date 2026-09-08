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

func (p *permissionUseCase) ListByRoleName(
	ctx context.Context,
	roleName string,
	params domain.PaginationParams,
	filters []domain.Filter,
) (*domain.PaginatedResult[domain.Permission], error) {
	if err := domain.Filters(filters).ValidateAllowed(allowedPermissionFilterFields); err != nil {
		return nil, err
	}

	if err := params.ValidateSort(allowedPermissionFilterFields); err != nil {
		return nil, err
	}

	result, err := p.permissionRepo.ListByRoleName(ctx, roleName, params, filters)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to list permissions by role name",
			err,
		)
	}

	return result, nil
}
