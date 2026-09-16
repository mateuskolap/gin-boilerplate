package usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
)

type permissionCheckerUseCase struct {
	rolePermissionRepo domain.RolePermissionRepository
	roleRepo           domain.RoleRepository
	cacheTTL           time.Duration
}

func NewPermissionCheckerUseCase(
	rolePermissionRepo domain.RolePermissionRepository,
	roleRepo domain.RoleRepository,
	cacheTTL time.Duration,
) domain.PermissionCheckerUseCase {
	return &permissionCheckerUseCase{
		rolePermissionRepo: rolePermissionRepo,
		roleRepo:           roleRepo,
		cacheTTL:           cacheTTL,
	}
}

func (p *permissionCheckerUseCase) HasPermission(ctx context.Context, roles []string, permission string) (bool, error) {
	if len(roles) == 0 {
		return false, nil
	}

	hasPerm, missingRoles, err := p.rolePermissionRepo.CheckRolesPermission(ctx, roles, permission)
	if err != nil {
		return p.hasPermissionFallbackDB(ctx, roles, permission)
	}

	if hasPerm {
		return true, nil
	}

	if len(missingRoles) == 0 {
		return false, nil
	}

	for _, role := range missingRoles {
		perms, err := p.loadAndCacheRolePermissions(ctx, role)
		if err != nil {
			roleObj, dbErr := p.roleRepo.GetByName(ctx, role, "Permissions")
			if dbErr == nil && roleObj != nil {
				for _, perm := range roleObj.Permissions {
					if perm.Name == permission {
						return true, nil
					}
				}
			}
			continue
		}

		if slices.Contains(perms, permission) {
			return true, nil
		}
	}

	return false, nil
}

func (p *permissionCheckerUseCase) loadAndCacheRolePermissions(ctx context.Context, role string) ([]string, error) {
	roleObj, err := p.roleRepo.GetByName(ctx, role, "Permissions")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role from DB: %w", err)
	}

	permissions := make([]string, 0)
	if roleObj != nil {
		for _, perm := range roleObj.Permissions {
			permissions = append(permissions, perm.Name)
		}
	}

	_ = p.rolePermissionRepo.SavePermissionsByRole(ctx, role, permissions, p.cacheTTL)

	return permissions, nil
}

func (p *permissionCheckerUseCase) hasPermissionFallbackDB(ctx context.Context, roles []string, permission string) (bool, error) {
	for _, role := range roles {
		roleObj, err := p.roleRepo.GetByName(ctx, role, "Permissions")
		if err != nil {
			return false, shared.NewAppError(
				shared.ErrTypeInternal,
				"Failed to check permissions from database fallback",
				err,
			)
		}

		if roleObj != nil {
			for _, perm := range roleObj.Permissions {
				if perm.Name == permission {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
