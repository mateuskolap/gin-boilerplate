package domain

import (
	"context"
	"time"
)

type RolePermissionRepository interface {
	SavePermissionsByRole(ctx context.Context, role string, permissions []string, ttl time.Duration) error
	ListPermissionsByRole(ctx context.Context, role string) ([]string, error)
	CheckRolesPermission(ctx context.Context, roles []string, permission string) (hasPermission bool, missingRoles []string, err error)
	InvalidatePermissionsByRole(ctx context.Context, role string) error
	InvalidateAll(ctx context.Context) error
}

type PermissionCheckerUseCase interface {
	// HasPermission checks if any of the provided roles has the specified permission.
	HasPermission(ctx context.Context, roles []string, permission string) (bool, error)
}
