package repository

import (
	"context"
	"errors"
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

func (r *roleRepository) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	var role domain.Role

	err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &role, nil
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
