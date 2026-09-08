package repository

import (
	"context"
	"gin-boilerplate/internal/domain"
	"time"
)

type rolePermissionRepository struct {
	cache domain.CacheRepository
}

func NewRolePermissionRepository(cache domain.CacheRepository) domain.RolePermissionRepository {
	return &rolePermissionRepository{
		cache: cache,
	}
}

func (r *rolePermissionRepository) FindPermissionsByRole(ctx context.Context, role string) ([]string, error) {
	panic("unimplemented")
}

func (r *rolePermissionRepository) InvalidatePermissionsByRole(ctx context.Context, role string) error {
	panic("unimplemented")
}

func (r *rolePermissionRepository) SavePermissionsByRole(ctx context.Context, role string, permissions []string, ttl time.Duration) error {
	panic("unimplemented")
}
