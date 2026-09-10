package repository

import (
	"context"
	"fmt"
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

func (r *rolePermissionRepository) SavePermissionsByRole(ctx context.Context, role string, permissions []string, ttl time.Duration) error {
	key := fmt.Sprintf("role_permissions:%s", role)
	return r.cache.Set(ctx, key, permissions, ttl)
}

func (r *rolePermissionRepository) ListPermissionsByRole(ctx context.Context, role string) ([]string, error) {
	key := fmt.Sprintf("role_permissions:%s", role)

	var permissions []string
	if err := r.cache.Get(ctx, key, &permissions); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (r *rolePermissionRepository) InvalidatePermissionsByRole(ctx context.Context, role string) error {
	key := fmt.Sprintf("role_permissions:%s", role)
	return r.cache.Delete(ctx, key)
}
