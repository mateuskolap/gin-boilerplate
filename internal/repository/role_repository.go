package repository

import (
	"context"
	"errors"
	"gin-boilerplate/internal/domain"

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
