package domain

import (
	"context"
	"time"
)

type RolePermissionRepository interface {
	// SavePermissionsByRole caches the permissions assigned to a role.
	SavePermissionsByRole(ctx context.Context, role string, permissions []string, ttl time.Duration) error

	// CheckRolesPermission verifies if any of the provided roles has the specified permission.
	// Returns true if permitted, or missingRoles indicating which roles had a cache miss.
	CheckRolesPermission(ctx context.Context, roles []string, permission string) (hasPermission bool, missingRoles []string, err error)

	// InvalidatePermissionsByRole evicts cached permissions for a specific role.
	InvalidatePermissionsByRole(ctx context.Context, role string) error

	// InvalidateAll evicts all cached role permissions.
	InvalidateAll(ctx context.Context) error
}

type PermissionCheckerUseCase interface {
	// HasPermission checks if any of the provided roles has the specified permission.
	HasPermission(ctx context.Context, roles []string, permission string) (bool, error)
}
