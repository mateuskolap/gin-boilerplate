package repository

import (
	"context"
	"gin-boilerplate/internal/domain"
	"uuid"

	"gorm.io/gorm"
)

type userRepository struct {
	*baseRepository[domain.User]
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{
		baseRepository: newBaseRepository[domain.User](db),
	}
}

func (r *userRepository) GetByEmail(ctx context.Context, email string, preloads ...string) (*domain.User, error) {
	return r.FindOneBy(ctx, "email = ?", []any{email}, preloads...)
}

func (r *userRepository) AddRoles(ctx context.Context, user domain.User, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	roles := make([]domain.Role, len(roleIDs))
	for i, id := range roleIDs {
		roles[i] = domain.Role{ID: id}
	}

	return r.getDB(ctx).Model(&user).Association("Roles").Append(&roles)
}

func (r *userRepository) RemoveRoles(ctx context.Context, user domain.User, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}

	roles := make([]domain.Role, len(roleIDs))
	for i, id := range roleIDs {
		roles[i] = domain.Role{ID: id}
	}

	return r.getDB(ctx).Model(&user).Association("Roles").Delete(&roles)
}
