package repository

import (
	"context"
	"fmt"
	"gin-boilerplate/internal/domain"
	"time"
)

const (
	rolePermissionKeyPrefix    = "role_permissions"
	rolePermissionVerKeyPrefix = "role_permissions_ver"
	emptyRoleSentinel          = "__EMPTY__"
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

func roleVersionKey(role string) string {
	return fmt.Sprintf("%s:%s", rolePermissionVerKeyPrefix, role)
}

func (r *rolePermissionRepository) GetRoleVersion(ctx context.Context, role string) (int64, error) {
	var version int64
	if err := r.cache.Get(ctx, roleVersionKey(role), &version); err != nil {
		return 0, err
	}
	return version, nil
}

func (r *rolePermissionRepository) SavePermissionsByRole(ctx context.Context, role string, permissions []string, ttl time.Duration, expectedVersion int64) (bool, error) {
	key := roleKey(role)
	versionKey := roleVersionKey(role)

	members := permissions
	if len(permissions) == 0 {
		members = []string{emptyRoleSentinel}
	}

	return r.cache.SetAddIfVersionMatch(ctx, key, members, ttl, versionKey, expectedVersion)
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
	if _, err := r.cache.Incr(ctx, roleVersionKey(role)); err != nil {
		return err
	}
	return r.cache.Delete(ctx, roleKey(role))
}

func (r *rolePermissionRepository) InvalidateAll(ctx context.Context) error {
	if err := r.cache.DeleteByPattern(ctx, rolePermissionKeyPrefix+":*"); err != nil {
		return err
	}
	return r.cache.DeleteByPattern(ctx, rolePermissionVerKeyPrefix+":*")
}
