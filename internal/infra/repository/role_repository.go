package repository

import (
	"context"
	"gin-boilerplate/internal/domain"

	"uuid"

	"gorm.io/gorm"
)

type roleRepository struct {
	domain.BaseRepository[domain.Role]
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) domain.RoleRepository {
	return &roleRepository{
		BaseRepository: NewBaseRepository[domain.Role](db),
		db:             db,
	}
}

func (r *roleRepository) GetByName(ctx context.Context, name string, preloads ...string) (*domain.Role, error) {
	return r.FindOneBy(ctx, "name = ?", []any{name}, preloads...)
}

func (r *roleRepository) AddPermissions(ctx context.Context, role domain.Role, permissionIDs []uuid.UUID) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	permissions := make([]domain.Permission, len(permissionIDs))
	for i, id := range permissionIDs {
		permissions[i] = domain.Permission{ID: id}
	}

	return r.db.WithContext(ctx).Model(&role).Association("Permissions").Append(&permissions)
}

func (r *roleRepository) RemovePermissions(ctx context.Context, role domain.Role, permissionIDs []uuid.UUID) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	permissions := make([]domain.Permission, len(permissionIDs))
	for i, id := range permissionIDs {
		permissions[i] = domain.Permission{ID: id}
	}

	return r.db.WithContext(ctx).Model(&role).Association("Permissions").Delete(&permissions)
}
