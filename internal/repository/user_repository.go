package repository

import (
	"context"
	"errors"
	"gin-boilerplate/internal/domain"

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
