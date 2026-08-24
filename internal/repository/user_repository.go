package repository

import (
	"context"
	"errors"
	"gin-boilerplate/internal/domain"
	"uuid"

	"gorm.io/gorm"
)

type userRepository struct {
	domain.BaseRepository[domain.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository[domain.User](db),
		db:             db,
	}
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *userRepository) AddRoles(ctx context.Context, user domain.User, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	roles := make([]domain.Role, len(roleIDs))
	for i, id := range roleIDs {
		roles[i] = domain.Role{ID: id}
	}

	return r.db.WithContext(ctx).Model(&user).Association("Roles").Append(&roles)
}

func (r *userRepository) RemoveRoles(ctx context.Context, user domain.User, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	roles := make([]domain.Role, len(roleIDs))
	for i, id := range roleIDs {
		roles[i] = domain.Role{ID: id}
	}

	return r.db.WithContext(ctx).Model(&user).Association("Roles").Delete(&roles)
}
