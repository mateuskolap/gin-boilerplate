package domain

import (
	"context"
	"time"
)

type RolePermissionRepository interface {
	FindPermissionsByRole(ctx context.Context, role string) ([]string, error)
	SavePermissionsByRole(ctx context.Context, role string, permissions []string, ttl time.Duration) error
	InvalidatePermissionsByRole(ctx context.Context, role string) error
}

type PermissionChecker interface {
	HasPermission(ctx context.Context, roles []string, permission string) (bool, error)
}
