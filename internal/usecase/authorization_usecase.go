package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"slices"
	"time"
)

type permissionCheckerUseCase struct {
	rolePermissionRepo domain.RolePermissionRepository
	roleRepo           domain.RoleRepository
}

func NewPermissionCheckerUseCase(
	rolePermissionRepo domain.RolePermissionRepository,
	roleRepo domain.RoleRepository,
) domain.PermissionCheckerUseCase {
	return &permissionCheckerUseCase{
		rolePermissionRepo: rolePermissionRepo,
		roleRepo:           roleRepo,
	}
}

func (p *permissionCheckerUseCase) HasPermission(ctx context.Context, roles []string, permission string) (bool, error) {
	for _, role := range roles {
		rolePermissions, err := p.rolePermissionRepo.ListPermissionsByRole(ctx, role)
		if err != nil {
			return false, domain.NewAppError(
				domain.ErrTypeInternal,
				"Failed to list permissions by role",
				err,
			)
		}

		if rolePermissions == nil {
			roleObj, err := p.roleRepo.GetByName(ctx, role, "Permissions")
			if err != nil {
				return false, domain.NewAppError(
					domain.ErrTypeInternal,
					"Failed to find role for permissions check",
					err,
				)
			}

			rolePermissions = make([]string, 0)
			if roleObj != nil {
				for _, perm := range roleObj.Permissions {
					rolePermissions = append(rolePermissions, perm.Name)
				}
			}

			_ = p.rolePermissionRepo.SavePermissionsByRole(ctx, role, rolePermissions, 24*time.Hour)
		}

		if slices.Contains(rolePermissions, permission) {
			return true, nil
		}
	}

	return false, nil
}
