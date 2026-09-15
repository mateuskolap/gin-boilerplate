package repository

import (
	"context"
	"fmt"
	"gin-boilerplate/internal/domain"
	"time"
)

const (
	rolePermissionKeyPrefix = "role_permissions"
	emptyRoleSentinel       = "__EMPTY__"
)

type rolePermissionRepository struct {
	cache domain.CacheRepository
}

func NewRolePermissionRepository(cache domain.CacheRepository) domain.RolePermissionRepository {
	return &rolePermissionRepository{
		cache: cache,
	}
}

func roleKey(role string) string {
	return fmt.Sprintf("%s:%s", rolePermissionKeyPrefix, role)
}

func (r *rolePermissionRepository) SavePermissionsByRole(ctx context.Context, role string, permissions []string, ttl time.Duration) error {
	key := roleKey(role)

	if len(permissions) == 0 {
		return r.cache.SetAdd(ctx, key, []string{emptyRoleSentinel}, ttl)
	}

	return r.cache.SetAdd(ctx, key, permissions, ttl)
}

func (r *rolePermissionRepository) CheckRolesPermission(ctx context.Context, roles []string, permission string) (bool, []string, error) {
	if len(roles) == 0 {
		return false, nil, nil
	}

	keys := make([]string, len(roles))
	keyToRole := make(map[string]string, len(roles))

	for i, role := range roles {
		k := roleKey(role)
		keys[i] = k
		keyToRole[k] = role
	}

	hasMember, missingKeys, err := r.cache.CheckSetMembers(ctx, keys, permission)
	if err != nil {
		return false, nil, err
	}

	if hasMember {
		return true, nil, nil
	}

	var missingRoles []string
	for _, k := range missingKeys {
		if role, ok := keyToRole[k]; ok {
			missingRoles = append(missingRoles, role)
		}
	}

	return false, missingRoles, nil
}

func (r *rolePermissionRepository) InvalidatePermissionsByRole(ctx context.Context, role string) error {
	return r.cache.Delete(ctx, roleKey(role))
}

func (r *rolePermissionRepository) InvalidateAll(ctx context.Context) error {
	return r.cache.DeleteByPattern(ctx, rolePermissionKeyPrefix+":*")
}
